package commands

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
		watch        bool
	)

	renderRows := func(rows []service.Row) {
		if len(rows) == 0 {
			logger.Info("No presets or managed containers found.")
			return
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
	}

	applyFilters := func(rows []service.Row) []service.Row {
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
		return filtered
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all presets and their container status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if watch {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
				for {
					rows, err := service.ListAll(cmd.Context())
					if err != nil {
						logger.Error(err.Error())
						return err
					}
					rows = applyFilters(rows)
					fmt.Print("\033[H\033[2J") // clear screen
					fmt.Printf("forge docker list  (refreshes every 2s — Ctrl+C to stop)  %s\n\n",
						time.Now().Format("15:04:05"))
					renderRows(rows)
					select {
					case <-sig:
						return nil
					case <-time.After(2 * time.Second):
					case <-cmd.Context().Done():
						return nil
					}
				}
			}

			rows, err := service.ListAll(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			rows = applyFilters(rows)

			if jsonOut {
				if len(rows) == 0 {
					return ui.EmitJSON([]service.Row{})
				}
				return ui.EmitJSON(rows)
			}

			renderRows(rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&filterEngine, "engine", "", "Filter by engine name (e.g. postgres, redis)")
	cmd.Flags().StringVar(&filterStatus, "status", "", "Filter by status (running, stopped, orphaned, legacy, not created)")
	cmd.Flags().BoolVar(&orphanedOnly, "orphaned", false, "Show only orphaned and legacy containers")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Refresh every 2 seconds (Ctrl+C to stop)")
	return cmd
}
