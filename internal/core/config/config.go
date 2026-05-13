package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Init sets viper defaults, reads ~/.forge/config.yaml, and writes it on first run.
func Init() {
	viper.SetDefault("docker.default_user", "forge")
	viper.SetDefault("docker.default_password", "forge_dev")
	viper.SetDefault("docker.default_db", "forge")
	viper.SetDefault("docker.port_range_start", 15000)
	viper.SetDefault("docker.port_range_end", 15999)
	viper.SetDefault("docker.readiness_timeout_seconds", 30)

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".forge", "config.yaml")
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
		_ = viper.WriteConfigAs(configPath)
	}
}

func DefaultUser() string          { return viper.GetString("docker.default_user") }
func DefaultPassword() string      { return viper.GetString("docker.default_password") }
func DefaultDB() string            { return viper.GetString("docker.default_db") }
func PortRangeStart() int          { return viper.GetInt("docker.port_range_start") }
func PortRangeEnd() int            { return viper.GetInt("docker.port_range_end") }
func ReadinessTimeoutSeconds() int { return viper.GetInt("docker.readiness_timeout_seconds") }

// EngineDefaultImage returns the user-configured default image for engineName, or "" if unset.
func EngineDefaultImage(engineName string) string {
	return viper.GetString("docker.engines." + engineName + ".default_image")
}

// Set writes key=value to ~/.forge/config.yaml. key must be a valid dotted viper key.
func Set(key, value string) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}
