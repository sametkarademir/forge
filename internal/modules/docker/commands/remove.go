package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

const (
	removeOptContainerVolume = "Remove container and volume (keep preset config)"
	removeOptAll             = "Remove container, volume, and preset config"
	removeOptCancel          = "Cancel"
)

func NewRemoveCommand() *cobra.Command {
	var yes, purge, purgeNetwork bool

	cmd := &cobra.Command{
		Use:   "remove <preset>",
		Short: "Remove a preset's container and volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			var mode service.RemoveMode
			switch {
			case purge:
				mode = service.RemoveAll
			case yes:
				mode = service.RemoveContainerVolume
			default:
				choice, err := ui.Select(
					fmt.Sprintf("How do you want to remove preset %q?", name),
					[]string{removeOptContainerVolume, removeOptAll, removeOptCancel},
					removeOptContainerVolume,
				)
				if err != nil || choice == removeOptCancel {
					logger.Info("Aborted.")
					return nil
				}
				if choice == removeOptAll {
					mode = service.RemoveAll
				} else {
					mode = service.RemoveContainerVolume
				}
			}

			if err := service.RemovePreset(cmd.Context(), name, mode); err != nil {
				logger.Error(err.Error())
				return err
			}

			if mode == service.RemoveAll {
				logger.Success(fmt.Sprintf("Removed preset %q and its container, volume, and config.", name))
			} else {
				logger.Success(fmt.Sprintf("Removed container and volume for preset %q.", name))
			}

			if purgeNetwork {
				if err := service.PruneNetwork(cmd.Context(), "forge-net"); err != nil {
					logger.Warn(err.Error())
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Remove container and volume, keep preset config")
	cmd.Flags().BoolVar(&purge, "purge", false, "Remove container, volume, and preset config")
	cmd.Flags().BoolVar(&purgeNetwork, "purge-network", false, "Also remove forge-net if no containers remain")
	return cmd
}
