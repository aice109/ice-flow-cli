package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/config"
	"github.com/aice109/ice-flow-cli/internal/i18n"
)

func (a *App) configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit configuration",
		Example: `  iceflow config
  iceflow config get lang
  iceflow config set lang zh-CN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printTable(nil, [][]string{
				{i18n.T("config"), config.ConfigPath()},
				{i18n.T("data"), config.DataDir()},
				{i18n.T("db"), config.DBPath()},
			})
			fmt.Println()
			keys := configKeys()
			rows := make([][]string, 0, len(keys))
			for _, k := range keys {
				v, _ := configGet(a.Cfg, k)
				rows = append(rows, []string{k, v})
			}
			printTable(nil, rows)
			return nil
		},
	}
	cmd.AddCommand(
		a.configPathCmd(),
		a.configGetCmd(),
		a.configSetCmd(),
	)
	return cmd
}

func (a *App) configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print config and data locations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(config.ConfigPath())
			fmt.Println(config.DBPath())
			return nil
		},
	}
}

func (a *App) configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, ok := configGet(a.Cfg, args[0])
			if !ok {
				return errors.New(i18n.Tf("unknown key %q (known: %s)", args[0], strings.Join(configKeys(), ", ")))
			}
			fmt.Println(v)
			return nil
		},
	}
}

func (a *App) configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !configSet(&a.Cfg, args[0], args[1]) {
				return errors.New(i18n.Tf("unknown key %q (known: %s)", args[0], strings.Join(configKeys(), ", ")))
			}
			if err := config.Save(a.Cfg); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("set %s = %s", args[0], args[1]))
			return nil
		},
	}
}

func configKeys() []string {
	return []string{
		"auto_update_check",
		"date_format",
		"default_priority",
		"github_repo",
		"lang",
		"notify",
		"notify_sound",
		"update_channel",
	}
}

func configGet(cfg config.Config, key string) (string, bool) {
	switch key {
	case "auto_update_check":
		return strconv.FormatBool(cfg.AutoUpdateCheck), true
	case "date_format":
		return cfg.DateFormat, true
	case "default_priority":
		return strconv.Itoa(cfg.DefaultPriority), true
	case "github_repo":
		return cfg.GitHubRepo, true
	case "lang":
		return cfg.Lang, true
	case "notify":
		return cfg.Notify, true
	case "notify_sound":
		return strconv.FormatBool(cfg.NotifySound), true
	case "update_channel":
		return cfg.UpdateChannel, true
	}
	return "", false
}

func configSet(cfg *config.Config, key, value string) bool {
	switch key {
	case "auto_update_check":
		cfg.AutoUpdateCheck = parseBool(value)
	case "date_format":
		cfg.DateFormat = value
	case "default_priority":
		if n, err := strconv.Atoi(value); err == nil {
			cfg.DefaultPriority = n
		}
	case "github_repo":
		cfg.GitHubRepo = value
	case "lang":
		cfg.Lang = value
	case "notify":
		cfg.Notify = value
	case "notify_sound":
		cfg.NotifySound = parseBool(value)
	case "update_channel":
		cfg.UpdateChannel = value
	default:
		return false
	}
	return true
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
