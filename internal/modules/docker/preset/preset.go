package preset

import "time"

// Preset is a named, reusable container blueprint stored at ~/.forge/presets/<name>.yaml.
// SchemaVersion 2 adds the Options field; v1 files load cleanly with Options == nil.
type Preset struct {
	SchemaVersion int               `yaml:"schema_version"`
	Name          string            `yaml:"name"`
	Engine        string            `yaml:"engine"`
	Image         string            `yaml:"image"`
	Database      string            `yaml:"database"`
	Username      string            `yaml:"username"`
	Password      string            `yaml:"password"`
	InternalPort  int               `yaml:"internal_port"`
	HostPort      int               `yaml:"host_port,omitempty"`
	Options       map[string]string `yaml:"options,omitempty"` // engine-specific extras; nil for DB engines
	CreatedAt     time.Time         `yaml:"created_at"`
}
