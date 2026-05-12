package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewRemoveCommand() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <project>",
		Short: "Remove a project's container, volume, and network membership",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := args[0]

			if !yes {
				ok, err := ui.Confirm(fmt.Sprintf("Remove project %q and all its data?", project))
				if err != nil || !ok {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := service.RemoveProject(cmd.Context(), project); err != nil {
				logger.Error(err.Error())
				return err
			}
			logger.Success(fmt.Sprintf("Removed project %q", project))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
