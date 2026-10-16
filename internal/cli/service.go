package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/service"
)

func (a *App) serviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage the background reminder job",
		Long: i18n.T("Install a resident background scheduler that fires reminders even when the CLI is closed.") + "\n\n" +
			i18n.T("macOS: launchd agent. Linux: systemd --user service (crontab fallback)."),
	}
	var install bool
	installCmd := &cobra.Command{
		Use:     "install",
		Short:   "Install the background reminder job",
		Example: "  iceflow service install",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			desc, err := service.Install(exe)
			if err != nil {
				return err
			}
			fmt.Println(desc)
			return nil
		},
	}
	installCmd.Flags().BoolVar(&install, "unused", false, "reserved")
	_ = installCmd.Flags().MarkHidden("unused")

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the background reminder job",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			desc, err := service.Uninstall()
			if err != nil {
				return err
			}
			fmt.Println(desc)
			return nil
		},
	}
	statusCmd := &cobra.Command{
		Use:     "status",
		Short:   "Show the background job status",
		Example: "  iceflow service status",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			desc, err := service.Status()
			if err != nil {
				return err
			}
			fmt.Println(desc)
			return nil
		},
	}

	cmd.AddCommand(installCmd, uninstallCmd, statusCmd)
	return cmd
}
