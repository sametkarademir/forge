package commands

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewResetCommand() *cobra.Command {
	var (
		yes        bool
		keepVolume bool
	)

	cmd := &cobra.Command{
		Use:   "reset <preset>",
		Short: "Wipe and recreate the database for a preset (same port and credentials)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if !yes {
				msg := fmt.Sprintf("This will DELETE all data for preset %q. Continue?", name)
				if keepVolume {
					msg = fmt.Sprintf("This will recreate the container for preset %q (volume kept). Continue?", name)
				}
				ok, err := ui.Confirm(msg)
				if err != nil || !ok {
					logger.Info("Aborted.")
					return nil
				}
			}

			if err := service.ResetPreset(cmd.Context(), name, service.ResetOptions{KeepVolume: keepVolume}); err != nil {
				var portErr *service.PortConflictError
				if errors.As(err, &portErr) {
					logger.Error(fmt.Sprintf("port %d is in use by another process:", portErr.Port))
					out, _ := exec.Command("lsof", "-i", fmt.Sprintf(":%d", portErr.Port)).Output()
					if len(out) > 0 {
						logger.Plain(string(out))
					}
					logger.Info(fmt.Sprintf("  Option A: stop the process on port %d, then re-run reset.", portErr.Port))
					logger.Info(fmt.Sprintf("  Option B: remove the preset and recreate it with a different port: forge docker remove %s --purge && forge docker create", name))
					return err
				}
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&keepVolume, "keep-volume", false, "Recreate container only, preserve data volume")
	return cmd
}
