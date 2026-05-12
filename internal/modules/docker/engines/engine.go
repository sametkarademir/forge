package engines

import (
	"fmt"
	"sort"
	"sync"
)

// Engine is implemented by each database engine file.
type Engine interface {
	Name() string
	DefaultImage() string
	DefaultPort() int
	DataDir() string // volume mount target inside the container
	EnvVars(user, password, db string) map[string]string
	ConnectionString(host string, hostPort int, user, password, db string) string
	ValidatePassword(password string) error
	PasswordEnvKey() string // env var key that holds the password in a running container
}

var (
	mu       sync.Mutex
	registry = map[string]Engine{}
)

// Register adds an engine to the global registry. Called from engine init() functions.
func Register(e Engine) {
	mu.Lock()
	defer mu.Unlock()
	registry[e.Name()] = e
}

// Get returns the engine for name, or false if not registered.
func Get(name string) (Engine, bool) {
	mu.Lock()
	defer mu.Unlock()
	e, ok := registry[name]
	return e, ok
}

// All returns all registered engines sorted by name.
func All() []Engine {
	mu.Lock()
	defer mu.Unlock()
	result := make([]Engine, 0, len(registry))
	for _, e := range registry {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name() < result[j].Name() })
	return result
}

// ErrUnknownEngine returns a user-friendly error for an unregistered engine name.
func ErrUnknownEngine(name string) error {
	return fmt.Errorf("unknown engine %q — run 'forge docker engines' to see supported engines", name)
}
