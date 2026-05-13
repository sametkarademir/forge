package commands

import (
	"strconv"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewShowCommand returns the command that shows preset config and container state.
func NewShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <preset>",
		Short: "Show preset configuration and container status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			view, err := service.ShowPreset(cmd.Context(), args[0])
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			p := view.Preset
			rows := [][]string{
				{"Name", p.Name},
				{"Engine", p.Engine},
				{"Image", p.Image},
				{"Username", p.Username},
				{"Password", "***"},
				{"Database", p.Database},
				{"Internal port", strconv.Itoa(p.InternalPort)},
				{"Created at", p.CreatedAt.Format("2006-01-02 15:04:05 UTC")},
				{"Status", view.Status},
			}
			if view.HostPort > 0 {
				rows = append(rows, []string{"Host port", strconv.Itoa(view.HostPort)})
			}
			if view.Primary != "" {
				rows = append(rows, []string{"Connection", view.Primary})
			}
			for _, ep := range view.Endpoints {
				rows = append(rows, []string{ep.Label, ep.Value})
			}

			ui.RenderTable([]string{"Setting", "Value"}, rows)
			return nil
		},
	}
}
