package commands

import (
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewConnCommand returns the connection-info command.
// By default it renders a table with the unmasked DSN and all engine endpoints (pretty mode).
// When stdout is not a TTY, or --raw is passed, it prints only the unmasked DSN via logger.Plain
// so the output is pipe-friendly (e.g. `forge docker conn mypreset | pbcopy`).
func NewConnCommand() *cobra.Command {
	var (
		raw     bool
		copy    bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "conn <preset>",
		Short: "Show connection info for a preset",
		Long: `Show connection info for a running preset container.

By default, prints a table with the connection string and any additional endpoints
(e.g. Management UI for RabbitMQ). When stdout is not a terminal, or --raw is
supplied, only the bare unmasked DSN is printed — suitable for piping:

  forge docker conn mypreset --raw | pbcopy
  forge docker conn mypreset --copy`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			view, err := service.GetConnView(cmd.Context(), args[0])
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			if copy {
				if err := ui.CopyToClipboard(view.Primary); err != nil {
					logger.Error(err.Error())
					return err
				}
				logger.Success("Connection string copied to clipboard.")
				return nil
			}

			if jsonOut {
				type jsonConn struct {
					Connection string            `json:"connection"`
					Endpoints  map[string]string `json:"endpoints,omitempty"`
				}
				eps := make(map[string]string, len(view.Endpoints))
				for _, ep := range view.Endpoints {
					eps[ep.Label] = ep.Value
				}
				out := jsonConn{Connection: view.Primary}
				if len(eps) > 0 {
					out.Endpoints = eps
				}
				return ui.EmitJSON(out)
			}

			if raw || !ui.IsInteractive() {
				logger.Plain(view.Primary)
				return nil
			}

			rows := [][]string{{"Connection", view.Primary}}
			for _, ep := range view.Endpoints {
				rows = append(rows, []string{ep.Label, ep.Value})
			}
			ui.RenderTable([]string{"Setting", "Value"}, rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print only the unmasked DSN (pipe-friendly)")
	cmd.Flags().BoolVarP(&copy, "copy", "c", false, "Copy the connection string to the clipboard (macOS)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
