package commands

import (
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/spf13/cobra"
)

func NewEnginesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "engines",
		Short: "List supported database engines and their default images",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			all := engines.All()
			rows := make([][]string, 0, len(all))
			for _, e := range all {
				rows = append(rows, []string{e.Name(), e.DefaultImage()})
			}
			ui.RenderTable([]string{"ENGINE", "DEFAULT IMAGE"}, rows)
			return nil
		},
	}
}
