package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/logger"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
	"github.com/spf13/cobra"
)

// NewShellCommand returns the command that opens an interactive shell inside
// a preset container using the engine's native CLI (psql, mysql, redis-cli…).
func NewShellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell <preset>",
		Short: "Open an interactive shell in a preset container (engine-specific CLI)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			p, err := preset.Load(name)
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			eng, ok := engines.Get(p.Engine)
			if !ok {
				return engines.ErrUnknownEngine(p.Engine)
			}

			sp, ok := eng.(engines.ShellProvider)
			if !ok {
				return fmt.Errorf("engine %q does not support an interactive shell", p.Engine)
			}

			dc, err := dockerclient.NewClient()
			if err != nil {
				logger.Error(err.Error())
				return err
			}

			c, err := dc.InspectByPreset(cmd.Context(), name)
			if err != nil {
				return fmt.Errorf("no container found for preset %q — start with: forge docker run %s", name, name)
			}
			if !c.State.Running {
				return fmt.Errorf("preset %q is not running — start with: forge docker run %s", name, name)
			}

			argv := sp.ShellCmd(p.Username, p.Password, p.Database)
			return dc.ExecInteractive(c.ID, argv)
		},
	}
}
