package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/config"
)

func (a *App) configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("config: %s\n", config.ConfigPath())
			fmt.Printf("data:   %s\n", config.DataDir())
			fmt.Printf("db:     %s\n\n", config.DBPath())
			keys := configKeys()
			for _, k := range keys {
				v, _ := configGet(a.Cfg, k)
				fmt.Printf("%-18s %s\n", k, v)
			}
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
				return fmt.Errorf("unknown key %q (known: %s)", args[0], strings.Join(configKeys(), ", "))
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
				return fmt.Errorf("unknown key %q (known: %s)", args[0], strings.Join(configKeys(), ", "))
			}
			if err := config.Save(a.Cfg); err != nil {
				return err
			}
			fmt.Printf("set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func configKeys() []string {
	return []string{
		"date_format",
		"default_priority",
		"notify",
		"notify_sound",
		"auto_update_check",
		"update_channel",
		"github_repo",
	}
}

func configGet(cfg config.Config, key string) (string, bool) {
	switch key {
	case "date_format":
		return cfg.DateFormat, true
	case "default_priority":
		return strconv.Itoa(cfg.DefaultPriority), true
	case "notify":
		return cfg.Notify, true
	case "notify_sound":
		return strconv.FormatBool(cfg.NotifySound), true
	case "auto_update_check":
		return strconv.FormatBool(cfg.AutoUpdateCheck), true
	case "update_channel":
		return cfg.UpdateChannel, true
	case "github_repo":
		return cfg.GitHubRepo, true
	}
	return "", false
}

func configSet(cfg *config.Config, key, value string) bool {
	switch key {
	case "date_format":
		cfg.DateFormat = value
	case "default_priority":
		if n, err := strconv.Atoi(value); err == nil {
			cfg.DefaultPriority = n
		}
	case "notify":
		cfg.Notify = value
	case "notify_sound":
		cfg.NotifySound = parseBool(value)
	case "auto_update_check":
		cfg.AutoUpdateCheck = parseBool(value)
	case "update_channel":
		cfg.UpdateChannel = value
	case "github_repo":
		cfg.GitHubRepo = value
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
