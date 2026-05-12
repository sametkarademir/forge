package commands

import (
	"fmt"
	"time"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all forge-managed database containers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			infos, err := service.ListProjects(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			if len(infos) == 0 {
				fmt.Println("No managed containers found.")
				return nil
			}

			rows := make([][]string, 0, len(infos))
			for _, info := range infos {
				rows = append(rows, []string{
					info.Name,
					info.Engine,
					info.Status,
					fmt.Sprintf("%d", info.HostPort),
					formatUptime(info.Uptime),
				})
			}
			ui.RenderTable([]string{"PROJECT", "ENGINE", "STATUS", "PORT", "UPTIME"}, rows)
			return nil
		},
	}
}

func formatUptime(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
