package registry

import (
	"sync"

	"github.com/spf13/cobra"
)

// Module is implemented by every forge feature module.
type Module interface {
	Name() string
	Command() *cobra.Command
}

var (
	mu      sync.Mutex
	modules []Module
)

// Register adds a module to the global registry. Called from module init() functions.
func Register(m Module) {
	mu.Lock()
	defer mu.Unlock()
	modules = append(modules, m)
}

// Commands returns the cobra.Command for every registered module.
func Commands() []*cobra.Command {
	mu.Lock()
	defer mu.Unlock()
	cmds := make([]*cobra.Command, 0, len(modules))
	for _, m := range modules {
		cmds = append(cmds, m.Command())
	}
	return cmds
}
