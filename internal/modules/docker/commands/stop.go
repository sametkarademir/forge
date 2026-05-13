package commands

import (
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewStopCommand returns the command that stops a preset's container.
func NewStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <preset>",
		Short: "Stop the container for a preset (idempotent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := service.StopPreset(cmd.Context(), args[0]); err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}
}
