package commands

import (
	"strconv"

	"github.com/fatih/color"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

var (
	statusGreen   = color.New(color.FgGreen).SprintFunc()
	statusYellow  = color.New(color.FgYellow).SprintFunc()
	statusRed     = color.New(color.FgRed).SprintFunc()
	statusMagenta = color.New(color.FgMagenta).SprintFunc()
	statusDim     = color.New(color.Faint).SprintFunc()
)

func colorStatus(s string) string {
	switch s {
	case "running":
		return statusGreen(s)
	case "stopped", "exited":
		return statusYellow(s)
	case "orphaned":
		return statusRed(s)
	case "legacy":
		return statusMagenta(s)
	default:
		return statusDim(s)
	}
}

// NewListCommand returns the command that lists all presets and their container status.
func NewListCommand() *cobra.Command {
	var (
		filterEngine string
		filterStatus string
		orphanedOnly bool
		jsonOut      bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all presets and their container status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows, err := service.ListAll(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			// Apply filters.
			filtered := rows[:0]
			for _, r := range rows {
				if filterEngine != "" && r.Engine != filterEngine {
					continue
				}
				if filterStatus != "" && r.Status != filterStatus {
					continue
				}
				if orphanedOnly && r.Status != "orphaned" && r.Status != "legacy" {
					continue
				}
				filtered = append(filtered, r)
			}
			rows = filtered

			if len(rows) == 0 {
				if jsonOut {
					return ui.EmitJSON([]service.Row{})
				}
				logger.Info("No presets or managed containers found.")
				return nil
			}

			if jsonOut {
				return ui.EmitJSON(rows)
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
					colorStatus(r.Status),
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

	cmd.Flags().StringVar(&filterEngine, "engine", "", "Filter by engine name (e.g. postgres, redis)")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by status (running, stopped, orphaned, legacy, not created)")
	cmd.Flags().BoolVar(&orphanedOnly, "orphaned", false, "Show only orphaned and legacy containers")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
