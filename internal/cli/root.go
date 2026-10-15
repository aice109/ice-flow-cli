// Package cli wires the cobra command tree together.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/store"
)

// App carries shared dependencies for the command tree.
type App struct {
	Cfg   config.Config
	Store *store.Store

	flagDB      string
	flagJSON    bool
	flagNoColor bool
	flagLang    string
}

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	locale, err := resolveLocale(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", buildinfo.AppName, err)
		return 1
	}
	i18n.SetLocale(locale)

	a := &App{}
	root := a.rootCmd()
	localizeTree(root)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", buildinfo.AppName, localizeError(err))
		return 1
	}
	if a.Store != nil {
		a.Store.Close()
	}
	return 0
}

// resolveLocale picks the output language before the command tree is built, so
// help text and flag descriptions are translated as well. Priority: the --lang
// flag, IFLOW_LANG, the config file, then the POSIX locale environment.
func resolveLocale(args []string) (string, error) {
	fallback := fallbackLocale()
	flagLang := scanLangFlag(args)
	if isAuto(flagLang) {
		return fallback, nil
	}
	if c := i18n.Canonical(flagLang); c != "" {
		return c, nil
	}
	// An explicit but unsupported value is a user error: report it in the
	// language we would otherwise have used.
	i18n.SetLocale(fallback)
	return "", fmt.Errorf("%s", i18n.Tf("unsupported language %q (supported: %s)",
		flagLang, strings.Join(i18n.Supported(), ", ")))
}

// fallbackLocale resolves the locale from everything except the --lang flag.
func fallbackLocale() string {
	if c := i18n.Canonical(os.Getenv("IFLOW_LANG")); c != "" {
		return c
	}
	if cfg, err := config.Load(); err == nil {
		if c := i18n.Canonical(cfg.Lang); c != "" {
			return c
		}
	}
	return i18n.Detect(os.Getenv)
}

// scanLangFlag finds --lang in the raw arguments, before cobra parses anything.
func scanLangFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--":
			return ""
		case strings.HasPrefix(a, "--lang="):
			return strings.TrimPrefix(a, "--lang=")
		case a == "--lang" && i+1 < len(args):
			return args[i+1]
		}
	}
	return ""
}

func isAuto(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return true
	}
	return false
}

func (a *App) rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   buildinfo.AppName,
		Short: i18n.T(buildinfo.Description),
		Long: buildinfo.AppName + " — " + i18n.T(buildinfo.Description) + "\n\n" +
			i18n.T("Tasks, reminders and a personal command dictionary, all stored locally.") + "\n" +
			i18n.T("Run without arguments for a quick overview of today."),
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
	root.SetUsageTemplate(usageTemplate)

	pf := root.PersistentFlags()
	pf.StringVar(&a.flagDB, "db", "", "path to the SQLite database (default: XDG data dir)")
	pf.BoolVar(&a.flagJSON, "json", false, "emit JSON instead of tables where supported")
	pf.BoolVar(&a.flagNoColor, "no-color", false, "disable ANSI colors")
	pf.StringVar(&a.flagLang, "lang", "auto", "language for output: auto, en, zh-CN")

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
	addGroups(root)
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
			printTable(nil, [][]string{
				{i18n.T("commit:"), buildinfo.Commit},
				{i18n.T("built:"), buildinfo.Date},
				{i18n.T("platform:"), buildinfo.Platform()},
				{i18n.T("home:"), buildinfo.RepoURL},
			})
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
				fmt.Println(i18n.Tf("created %s", cfgPath))
			} else {
				fmt.Println(i18n.Tf("config already exists %s", cfgPath))
			}
			fmt.Println(i18n.Tf("database ready at %s", config.DBPath()))
			return nil
		},
	}
}
