package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewPruneCommand returns the command that removes orphaned and legacy managed resources.
func NewPruneCommand() *cobra.Command {
	var (
		yes         bool
		withNetwork bool
	)

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove orphaned and legacy forge-managed containers and volumes",
		Long: `Find and remove forge-managed containers and volumes that have no matching preset file.

  Orphaned  — container with forge.preset label but the preset YAML is gone.
  Legacy    — container without forge.preset label (pre-v2 format).
  Dangling  — volume with forge.managed=true but no matching container or preset.

Use --network to also remove forge-net when no containers are connected.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			targets, err := service.FindOrphans(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			if len(targets) == 0 && !withNetwork {
				logger.Info("Nothing to prune.")
				return nil
			}

			if len(targets) > 0 {
				logger.Info(fmt.Sprintf("Found %d resource(s) to remove:", len(targets)))
				for _, t := range targets {
					logger.Info(fmt.Sprintf("  %-12s %s  (%s)", t.Kind, t.Name, t.Reason))
				}
				logger.Info("")

				if !yes {
					ok, err := ui.Confirm(fmt.Sprintf("Remove %d resource(s)?", len(targets)))
					if err != nil || !ok {
						logger.Info("Aborted.")
						return nil
					}
				}

				if err := service.Prune(cmd.Context(), targets); err != nil {
					logger.Error(err.Error())
					return err
				}
			}

			if withNetwork {
				if err := service.PruneNetwork(cmd.Context(), "forge-net"); err != nil {
					logger.Error(err.Error())
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&withNetwork, "network", false, "Also remove forge-net if no containers are connected")
	return cmd
}
