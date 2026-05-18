package commands

import (
	"io"
	"os"
	"strconv"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sametkarademir/forge/internal/core/logger"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewLogsCommand returns the command that streams container logs for a preset.
func NewLogsCommand() *cobra.Command {
	var (
		follow bool
		tail   int
		since  string
	)

	cmd := &cobra.Command{
		Use:   "logs <preset>",
		Short: "Stream container logs for a preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tailStr := ""
			if tail > 0 {
				tailStr = strconv.Itoa(tail)
			}
			rc, err := service.LogsPreset(cmd.Context(), args[0], dockerclient.LogsOptions{
				Follow: follow,
				Tail:   tailStr,
				Since:  since,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			defer func() { _ = rc.Close() }()
			// ContainerLogs returns a multiplexed stream; StdCopy demultiplexes it.
			if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, rc); err != nil && err != io.EOF {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "n", 0, "Number of lines to show from the end (0 = all)")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since a duration (e.g. 5m, 1h) or RFC3339 timestamp")
	return cmd
}
