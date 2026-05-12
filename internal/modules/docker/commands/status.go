package commands

import (
	"fmt"
	"time"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <project>",
		Short: "Show detailed info for a project's database container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := service.GetProjectStatus(cmd.Context(), args[0])
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			fmt.Printf("Project:   %s\n", info.Name)
			fmt.Printf("Engine:    %s\n", info.Engine)
			fmt.Printf("Status:    %s\n", info.Status)
			fmt.Printf("Image:     %s\n", info.Image)
			fmt.Printf("Port:      %d\n", info.HostPort)
			fmt.Printf("Volume:    %s\n", info.VolumeName)
			fmt.Printf("Created:   %s\n", info.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Uptime:    %s\n", formatUptime(info.Uptime))
			fmt.Println()
			fmt.Println("Environment:")
			for k, v := range info.EnvSummary {
				fmt.Printf("  %-28s %s\n", k, v)
			}
			fmt.Println()
			fmt.Printf("Connection: %s\n", info.ConnectionString)
			return nil
		},
	}
}
