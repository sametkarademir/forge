package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
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
		quiet   bool
		format  string
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

			// --format: use FormatProvider if engine supports it.
			if format != "" {
				p, err := preset.Load(args[0])
				if err != nil {
					logger.Error(err.Error())
					return err
				}
				eng, ok := engines.Get(p.Engine)
				if !ok {
					return engines.ErrUnknownEngine(p.Engine)
				}
				fp, ok := eng.(engines.FormatProvider)
				if !ok {
					return fmt.Errorf("engine %q does not support --format (only primary DSN available)", p.Engine)
				}
				hostPort, err := service.GetPresetHostPort(cmd.Context(), args[0])
				if err != nil {
					logger.Error(err.Error())
					return err
				}
				formats := fp.ConnectionFormats(engines.ConnArgs{
					Host:     "localhost",
					HostPort: hostPort,
					User:     p.Username,
					Password: p.Password,
					Database: p.Database,
					Options:  p.Options,
				})
				if val, ok := formats[strings.ToLower(format)]; ok {
					logger.Plain(val)
					return nil
				}
				keys := make([]string, 0, len(formats))
				for k := range formats {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				return fmt.Errorf("unknown format %q — available for %s: %s", format, p.Engine, strings.Join(keys, ", "))
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

			if raw || quiet || !ui.IsInteractive() {
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
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Print only the DSN, no decoration (alias for --raw)")
	cmd.Flags().StringVar(&format, "format", "", "Output a specific connection-string format (e.g. uri, jdbc, psql, cli, ado, sqlcmd)")
	return cmd
}
