package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/update"
)

func (a *App) updateCmd() *cobra.Command {
	var check, force bool
	var repo string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update iceflow to the latest release",
		Long:  i18n.T("Download and install the latest release from GitHub, verifying its checksum before replacing the running binary."),
		Example: `  iceflow update --check
  iceflow update`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if repo == "" {
				repo = a.Cfg.GitHubRepo
			}
			if repo == "" {
				repo = buildinfo.RepoSlug
			}
			client := update.NewClient(repo)
			latest, hasUpdate, rel, err := client.Check(buildinfo.Version)
			if err != nil {
				return err
			}
			if !hasUpdate && !force {
				fmt.Println(i18n.Tf("iceflow is up to date (%s)", buildinfo.Version))
				return nil
			}
			fmt.Println(i18n.Tf("current %s → latest %s", buildinfo.Version, latest))
			if check {
				fmt.Println(i18n.T("run `iceflow update` to install"))
				return nil
			}
			path, err := client.Apply(rel)
			if err != nil {
				if errors.Is(err, fs.ErrPermission) {
					return fmt.Errorf("%w\n%s", err,
						i18n.T("hint: the binary is not writable; reinstall via your package manager or move iceflow to a user-writable directory"))
				}
				return err
			}
			fmt.Println(i18n.Tf("updated to %s (%s)", latest, path))
			if rel.Body != "" {
				fmt.Println()
				fmt.Println(rel.Body)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only check for a newer version")
	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already current")
	cmd.Flags().StringVar(&repo, "repo", "", "override the GitHub owner/repo")
	return cmd
}

// maybeAutoUpdateCheck performs a throttled (once a day) background check and
// prints a one-line notice when a newer release exists. Failures are silent.
func (a *App) maybeAutoUpdateCheck(cmd *cobra.Command) {
	if !a.Cfg.AutoUpdateCheck || os.Getenv("ICEFLOW_NO_UPDATE_CHECK") != "" {
		return
	}
	if buildinfo.Version == "dev" {
		return
	}
	switch cmd.Name() {
	case "update", "version", "completion", "help":
		return
	}
	cache := config.StatePath("update-check")
	if data, err := os.ReadFile(cache); err == nil {
		if t, err := time.Parse(time.RFC3339, string(data)); err == nil && time.Since(t) < 24*time.Hour {
			return
		}
	}
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(cache, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)

	client := update.NewClient(a.Cfg.GitHubRepo)
	client.HTTP.Timeout = 3 * time.Second
	latest, hasUpdate, _, err := client.Check(buildinfo.Version)
	if err != nil || !hasUpdate {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s\n", i18n.Tf("%s %s is available (you have %s). Run `iceflow update`.",
		buildinfo.AppName, latest, buildinfo.Version))
}
