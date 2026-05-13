package commands

import (
	"strconv"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewListCommand returns the command that lists all presets and their container status.
func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all presets and their container status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows, err := service.ListAll(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			if len(rows) == 0 {
				logger.Info("No presets or managed containers found.")
				return nil
			}

			tableRows := make([][]string, 0, len(rows))
			for _, r := range rows {
				port := ""
				if r.HostPort > 0 {
					port = strconv.Itoa(r.HostPort)
				}
				created := ""
				if !r.CreatedAt.IsZero() {
					created = r.CreatedAt.Format("2006-01-02")
				}
				tableRows = append(tableRows, []string{
					r.Name,
					r.Engine,
					r.Image,
					port,
					r.Status,
					created,
				})
			}

			ui.RenderTable(
				[]string{"PRESET", "ENGINE", "IMAGE", "PORT", "STATUS", "CREATED"},
				tableRows,
			)
			return nil
		},
	}
}
