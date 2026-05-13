package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewRunCommand returns the command that creates/starts a preset container.
func NewRunCommand() *cobra.Command {
	var (
		noWait  bool
		timeout int
	)

	cmd := &cobra.Command{
		Use:   "run <preset>",
		Short: "Create and start a container from a preset (idempotent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := service.RunPreset(cmd.Context(), args[0], service.RunOptions{
				NoWait:  noWait,
				Timeout: timeout,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			logger.Success(fmt.Sprintf("Running %s on port %d", args[0], info.HostPort))
			if info.ConnectionString != "" {
				logger.Info("  Connection: " + info.ConnectionString)
			}
			for _, ep := range info.Endpoints {
				logger.Info("  " + ep.Label + ": " + ep.Value)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Skip waiting for the DB to accept connections")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Readiness timeout in seconds (default: from config)")
	return cmd
}
