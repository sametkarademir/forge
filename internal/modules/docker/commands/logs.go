package commands

import (
	"io"
	"os"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewLogsCommand returns the command that streams container logs for a preset.
func NewLogsCommand() *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs <preset>",
		Short: "Stream container logs for a preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := service.LogsPreset(cmd.Context(), args[0], follow)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			defer rc.Close()
			// ContainerLogs returns a multiplexed stream; StdCopy demultiplexes it.
			if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, rc); err != nil && err != io.EOF {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	return cmd
}
