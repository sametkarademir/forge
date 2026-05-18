package engines

import (
	"fmt"
	"sort"
	"sync"
)

// ConnArgs is the runtime context an engine needs to produce its connection info.
// Options may be nil for engines that do not require extra configuration.
// ExtraPorts holds host ports allocated for additional container ports (OptionKey → host port);
// populated only for engines that implement ExtraPortProvider.
type ConnArgs struct {
	Host       string
	HostPort   int
	User       string
	Password   string
	Database   string
	Options    map[string]string // engine-specific extras from Preset.Options; may be nil
	ExtraPorts map[string]int    // OptionKey → assigned host port; nil for single-port engines
}

// ExtraPort declares an additional container port an engine exposes beyond its primary port.
// OptionKey is the key under Preset.Options where the user's preferred host port is stored.
type ExtraPort struct {
	Label         string // human label, e.g. "Management UI"
	ContainerPort int    // fixed port inside the container (e.g. 15672)
	OptionKey     string // e.g. "mgmt_host_port"; used as label suffix and Options key
}

// OptionPrompt describes one engine-specific question the create wizard should ask.
// The answer is stored under Key in Preset.Options.
type OptionPrompt struct {
	Key      string
	Label    string
	Default  string
	Validate func(string) error // may be nil
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
	// Cmd returns the container command override. Most engines return nil (use the image's
	// default ENTRYPOINT/CMD). Engines that configure auth or runtime options via CLI flags
	// (e.g. Redis --requirepass) return the full argv here.
	Cmd(password string) []string
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

// ExtraPortProvider is implemented by engines that expose more than one host port.
// Engines opt in by implementing this interface; single-port engines do not need it.
// image is the resolved Docker image tag (used to gate ports on image variants, e.g. -management).
// opts is Preset.Options, providing user-configured preferred host ports.
type ExtraPortProvider interface {
	Engine
	ExtraPorts(image string, opts map[string]string) []ExtraPort
}

// WizardPromptProvider is implemented by engines that require engine-specific questions
// beyond the standard wizard prompts (user, password, database, host port).
// image is the resolved Docker image tag, available for conditional prompts.
type WizardPromptProvider interface {
	Engine
	WizardPrompts(image string) []OptionPrompt
}

// ShellProvider is implemented by engines that support an interactive shell
// command (psql, mysql, redis-cli, etc.). The returned argv is run inside
// the container via `docker exec -it`.
// user and db may be empty strings if the engine doesn't require them.
type ShellProvider interface {
	Engine
	ShellCmd(user, password, db string) []string
}
