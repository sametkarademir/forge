package docker

import (
	"github.com/sametkarademir/forge/internal/core/registry"
	"github.com/sametkarademir/forge/internal/modules/docker/commands"
	"github.com/spf13/cobra"
)

type dockerModule struct{}

func init() {
	registry.Register(&dockerModule{})
}

func (m *dockerModule) Name() string { return "docker" }

func (m *dockerModule) Command() *cobra.Command {
	root := &cobra.Command{
		Use:   "docker",
		Short: "Manage database container presets",
	}

	root.AddCommand(commands.NewCreateCommand())
	root.AddCommand(commands.NewRunCommand())
	root.AddCommand(commands.NewStopCommand())
	root.AddCommand(commands.NewShowCommand())
	root.AddCommand(commands.NewListCommand())
	root.AddCommand(commands.NewConnCommand())
	root.AddCommand(commands.NewLogsCommand())
	root.AddCommand(commands.NewResetCommand())
	root.AddCommand(commands.NewRemoveCommand())
	root.AddCommand(commands.NewPruneCommand())
	root.AddCommand(commands.NewDoctorCommand())
	root.AddCommand(commands.NewEnginesCommand())

	return root
}
