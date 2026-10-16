package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/service"
	"github.com/aice109/ice-flow-cli/internal/store"
)

// Archive member names. Restore only looks for these, so unknown entries in a
// tar are ignored (and path traversal is impossible).
const (
	backupConfigName = "config.toml"
	backupDBName     = "iceflow.db"
	// backupDBLegacyName is the database member produced before the rename;
	// restore accepts it so archives from the old `iflow` binary still work.
	backupDBLegacyName = "iflow.db"
	backupManifestName = "manifest.json"
)

type backupManifest struct {
	App       string
	Version   string
	CreatedAt string
	Platform  string
	Files     []string
}

func (a *App) backupCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "backup [file]",
		Short: "Back up configuration and database",
		Long: i18n.T("Writes config.toml and a consistent snapshot of the SQLite database into a single .tar.gz archive.") + "\n\n" +
			i18n.T("The snapshot is taken with SQLite's VACUUM INTO, so it is safe to run while the background scheduler is active."),
		Example: `  iceflow backup
  iceflow backup ~/iceflow-backup-2026-10-16.tar.gz
  iceflow backup -o /Volumes/usb/iceflow.tar.gz`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := output
			if len(args) == 1 {
				if output != "" {
					return errors.New(i18n.T("pass either a positional path or --output, not both"))
				}
				dest = args[0]
			}
			if dest == "" {
				dest = fmt.Sprintf("iceflow-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
			}
			if err := a.writeBackup(dest); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("backed up to %s", dest))
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "destination archive (default: ./iceflow-backup-<timestamp>.tar.gz)")
	return cmd
}

// writeBackup builds a .tar.gz with the config file and a consistent database
// snapshot. It is also used to stash the current state before a restore.
func (a *App) writeBackup(dest string) error {
	tmp, err := os.MkdirTemp("", "iceflow-backup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	var files []string
	if fileExists(config.ConfigPath()) {
		if err := copyFile(config.ConfigPath(), filepath.Join(tmp, backupConfigName)); err != nil {
			return err
		}
		files = append(files, backupConfigName)
	}
	dbPath := a.databasePath()
	if fileExists(dbPath) {
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		err = s.Backup(filepath.Join(tmp, backupDBName))
		s.Close()
		if err != nil {
			return err
		}
		files = append(files, backupDBName)
	}
	if len(files) == 0 {
		return errors.New(i18n.T("nothing to back up (no config or database found)"))
	}

	manifest := backupManifest{
		App:       buildinfo.AppName,
		Version:   buildinfo.Version,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Platform:  buildinfo.Platform(),
		Files:     files,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, backupManifestName), data, 0o644); err != nil {
		return err
	}
	return writeTarGz(dest, tmp, append([]string{backupManifestName}, files...))
}

func (a *App) restoreCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "restore <archive>",
		Short: "Restore configuration and database from a backup",
		Long: i18n.T("Replaces config.toml and the SQLite database with the contents of an archive created by `iceflow backup`.") + "\n\n" +
			i18n.T("The current state is saved to a pre-restore archive first, and the background scheduler is restarted so it reopens the restored database."),
		Example: `  iceflow restore iceflow-backup-2026-10-16.tar.gz
  iceflow restore backup.tar.gz --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			archive := args[0]
			tmp, err := os.MkdirTemp("", "iceflow-restore-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			if err := extractTarGz(archive, tmp); err != nil {
				return fmt.Errorf("%s: %w", archive, err)
			}
			configSrc := filepath.Join(tmp, backupConfigName)
			dbSrc := filepath.Join(tmp, backupDBName)
			hasConfig, hasDB := fileExists(configSrc), fileExists(dbSrc)
			if !hasConfig && !hasDB {
				return errors.New(i18n.Tf("%s does not look like an iceflow backup", archive))
			}

			if !yes && !confirm(i18n.T("Restore will overwrite the current configuration and database. Continue?")) {
				return errors.New(i18n.T("aborted (use --yes to skip confirmation)"))
			}

			// Keep a copy of what we are about to replace.
			pre := fmt.Sprintf("iceflow-pre-restore-%s.tar.gz", time.Now().Format("20060102-150405"))
			if err := a.writeBackup(pre); err == nil {
				fmt.Println(i18n.Tf("current state saved to %s", pre))
			}

			if a.Store != nil {
				a.Store.Close()
				a.Store = nil
			}

			if hasConfig {
				if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o755); err != nil {
					return err
				}
				if err := copyFile(configSrc, config.ConfigPath()); err != nil {
					return err
				}
				fmt.Println(i18n.Tf("restored %s", config.ConfigPath()))
			}
			if hasDB {
				dbPath := a.databasePath()
				if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
					return err
				}
				// Drop stale WAL/SHM journals so the restored file is authoritative.
				_ = os.Remove(dbPath + "-wal")
				_ = os.Remove(dbPath + "-shm")
				if err := copyFile(dbSrc, dbPath); err != nil {
					return err
				}
				fmt.Println(i18n.Tf("restored %s", dbPath))
			}

			if service.Installed() {
				if desc, err := service.Restart(); err != nil {
					fmt.Fprintln(os.Stderr, i18n.Tf("could not restart the background scheduler: %v", err))
				} else if desc != "" {
					fmt.Println(desc)
				}
			} else {
				fmt.Println(i18n.T("restart the background scheduler with `iceflow service install` if you use one"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

// --- tar helpers ----------------------------------------------------------

func writeTarGz(dest, dir string, names []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, name := range names {
		if err := addFileToTar(tw, filepath.Join(dir, name), name); err != nil {
			tw.Close()
			gz.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func addFileToTar(tw *tar.Writer, path, name string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

// extractTarGz writes the recognized members into dir. Only the base name is
// used, so entries like ../etc/passwd are ignored rather than escaping dir.
func extractTarGz(src, dir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Base(hdr.Name)
		target := name
		switch name {
		case backupConfigName, backupDBName:
		case backupDBLegacyName:
			target = backupDBName
		default:
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.Create(filepath.Join(dir, target))
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
