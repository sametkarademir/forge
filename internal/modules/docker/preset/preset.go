package preset

import "time"

// Preset is a named, reusable database container blueprint stored at
// ~/.forge/presets/<name>.yaml.
type Preset struct {
	SchemaVersion int       `yaml:"schema_version"`
	Name          string    `yaml:"name"`
	Engine        string    `yaml:"engine"`
	Image         string    `yaml:"image"`
	Database      string    `yaml:"database"`
	Username      string    `yaml:"username"`
	Password      string    `yaml:"password"`
	InternalPort  int       `yaml:"internal_port"`
	CreatedAt     time.Time `yaml:"created_at"`
}
