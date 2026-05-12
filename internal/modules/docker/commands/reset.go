package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewResetCommand() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset <project>",
		Short: "Wipe and recreate the database for a project (same port and credentials)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := args[0]

			if !yes {
				ok, err := ui.Confirm(fmt.Sprintf("This will DELETE all data for project %q. Continue?", project))
				if err != nil || !ok {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := service.ResetProject(cmd.Context(), project); err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}
