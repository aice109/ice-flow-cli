// Package cli wires the cobra command tree together.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/store"
)

// App carries shared dependencies for the command tree.
type App struct {
	Cfg   config.Config
	Store *store.Store

	flagDB      string
	flagJSON    bool
	flagNoColor bool
}

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	a := &App{}
	root := a.rootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", buildinfo.AppName, err)
		return 1
	}
	if a.Store != nil {
		a.Store.Close()
	}
	return 0
}

func (a *App) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   buildinfo.AppName,
		Short: buildinfo.Description,
		Long: buildinfo.AppName + " — " + buildinfo.Description + `

Tasks, reminders and a personal command dictionary, all stored locally.
Run without arguments for a quick overview of today.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			a.setupColor()
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			a.Cfg = cfg
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runToday(cmd)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			a.maybeAutoUpdateCheck(cmd)
			return nil
		},
	}
	pf := root.PersistentFlags()
	pf.StringVar(&a.flagDB, "db", "", "path to the SQLite database (default: XDG data dir)")
	pf.BoolVar(&a.flagJSON, "json", false, "emit JSON instead of tables where supported")
	pf.BoolVar(&a.flagNoColor, "no-color", false, "disable ANSI colors")

	root.AddCommand(
		a.versionCmd(),
		a.initCmd(),
		a.taskCmd(),
		a.remindCmd(),
		a.snippetCmd(),
		a.findCmd(),
		a.runCmd(),
		a.updateCmd(),
		a.serviceCmd(),
		a.configCmd(),
	)
	return root
}

func (a *App) setupColor() {
	colorEnabled = !a.flagNoColor && os.Getenv("NO_COLOR") == "" && isTerminal(os.Stdout)
}

// openStore lazily opens the database.
func (a *App) openStore() (*store.Store, error) {
	if a.Store != nil {
		return a.Store, nil
	}
	path := a.flagDB
	if path == "" {
		path = config.DBPath()
	}
	s, err := store.Open(path)
	if err != nil {
		return nil, err
	}
	a.Store = s
	return s, nil
}

func (a *App) versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.flagJSON {
				return printJSON(map[string]string{
					"version":  buildinfo.Version,
					"commit":   buildinfo.Commit,
					"date":     buildinfo.Date,
					"platform": buildinfo.Platform(),
				})
			}
			fmt.Printf("%s %s\n", buildinfo.AppName, buildinfo.Version)
			fmt.Printf("  commit:   %s\n", buildinfo.Commit)
			fmt.Printf("  built:    %s\n", buildinfo.Date)
			fmt.Printf("  platform: %s\n", buildinfo.Platform())
			fmt.Printf("  home:     %s\n", buildinfo.RepoURL)
			return nil
		},
	}
}

func (a *App) initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the config file and database",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			created, err := config.EnsureInit()
			if err != nil {
				return err
			}
			if _, err := a.openStore(); err != nil {
				return err
			}
			cfgPath := config.ConfigPath()
			if created {
				fmt.Println("created " + cfgPath)
			} else {
				fmt.Println("config already exists " + cfgPath)
			}
			fmt.Println("database ready at " + config.DBPath())
			return nil
		},
	}
}
