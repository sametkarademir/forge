package commands

import (
	"fmt"
	"strconv"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewShowCommand returns the command that shows preset config and container state.
func NewShowCommand() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <preset>",
		Short: "Show preset configuration and container status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			view, err := service.ShowPreset(cmd.Context(), args[0])
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			if jsonOut {
				// Password masked in JSON output.
				type jsonView struct {
					Name         string `json:"name"`
					Engine       string `json:"engine"`
					Image        string `json:"image"`
					Username     string `json:"username"`
					Database     string `json:"database"`
					InternalPort int    `json:"internal_port"`
					HostPort     int    `json:"host_port,omitempty"`
					Status       string `json:"status"`
					Connection   string `json:"connection,omitempty"`
				}
				p := view.Preset
				return ui.EmitJSON(jsonView{
					Name:         p.Name,
					Engine:       p.Engine,
					Image:        p.Image,
					Username:     p.Username,
					Database:     p.Database,
					InternalPort: p.InternalPort,
					HostPort:     view.HostPort,
					Status:       view.Status,
					Connection:   view.Primary,
				})
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

			switch view.Status {
			case "not created":
				logger.Info(fmt.Sprintf("→ Start with: forge docker run %s", args[0]))
			case "exited", "stopped":
				logger.Info(fmt.Sprintf("→ Resume with: forge docker run %s", args[0]))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON (password masked)")
	return cmd
}
