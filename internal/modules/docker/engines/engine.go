package engines

import (
	"fmt"
	"sort"
	"sync"
)

// ConnArgs is the runtime context an engine needs to produce its connection info.
// Options may be nil for engines that do not require extra configuration.
type ConnArgs struct {
	Host     string
	HostPort int
	User     string
	Password string
	Database string
	Options  map[string]string // engine-specific extras from Preset.Options; may be nil
}

// Endpoint is one named additional connection endpoint.
// Values must not contain secrets — the engine keeps passwords out of these.
type Endpoint struct {
	Label string // e.g. "Management UI", "AMQP", "CLI"
	Value string
}

// ConnInfo is what an engine returns for display/copy purposes.
// Primary is the single-line canonical form used by `conn` (pipe-friendly).
// MaskedPrimary is the same string with the password obscured, used by `show`.
// Endpoints are additional rows shown by `show` and `run`; nil for DB engines.
type ConnInfo struct {
	Primary       string
	MaskedPrimary string
	Endpoints     []Endpoint
}

// Engine is implemented by each engine file (self-registered via init).
type Engine interface {
	Name() string
	DefaultImage() string
	ImageRepos() []string // Docker Hub / registry repo names, for filtering local images
	DefaultPort() int
	DataDir(image string) string // volume mount target inside the container
	EnvVars(user, password, db string) map[string]string
	ConnectionInfo(args ConnArgs) ConnInfo
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
