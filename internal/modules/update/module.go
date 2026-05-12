package update

import (
	"github.com/spf13/cobra"

	"github.com/sametkarademir/forge/internal/core/registry"
	"github.com/sametkarademir/forge/internal/modules/update/commands"
)

type updateModule struct{}

func init() {
	registry.Register(&updateModule{})
}

func (m *updateModule) Name() string { return "update" }

func (m *updateModule) Command() *cobra.Command {
	return commands.NewUpdateCommand()
}
