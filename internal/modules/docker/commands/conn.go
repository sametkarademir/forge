package commands

import (
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewConnCommand returns the pipe-friendly command that prints only the DSN.
func NewConnCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "conn <preset>",
		Short: "Print the connection string for a preset (pipe-friendly)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn, err := service.ConnString(cmd.Context(), args[0])
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			logger.Plain(dsn)
			return nil
		},
	}
}
