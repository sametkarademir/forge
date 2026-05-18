package commands

import (
	"fmt"
	"strconv"
	"strings"

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
				repos := strings.Join(e.ImageRepos(), ", ")
				port := strconv.Itoa(e.DefaultPort())
				extra := ""
				if epp, ok := e.(engines.ExtraPortProvider); ok {
					var parts []string
					for _, ep := range epp.ExtraPorts(e.DefaultImage(), nil) {
						parts = append(parts, fmt.Sprintf("%s (%d)", ep.Label, ep.ContainerPort))
					}
					extra = strings.Join(parts, ", ")
				}
				rows = append(rows, []string{e.Name(), e.DefaultImage(), repos, port, extra})
			}
			ui.RenderTable([]string{"ENGINE", "DEFAULT IMAGE", "IMAGE REPOS", "PORT", "EXTRA PORTS"}, rows)
			return nil
		},
	}
}
