package commands

import (
	"fmt"
	"strings"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/spf13/cobra"
)

// dockerSettings builds the authoritative list of configurable docker keys.
// Called at command-construction time so engine registry is fully populated.
func dockerSettings() []config.Setting {
	settings := []config.Setting{
		{Key: "docker.default_user", DefaultValue: config.DefaultUser},
		{Key: "docker.default_password", DefaultValue: config.DefaultPassword},
		{Key: "docker.default_db", DefaultValue: config.DefaultDB},
		{
			Key: "docker.port_range_start",
			DefaultValue: func() string {
				return fmt.Sprintf("%d", config.PortRangeStart())
			},
		},
		{
			Key: "docker.port_range_end",
			DefaultValue: func() string {
				return fmt.Sprintf("%d", config.PortRangeEnd())
			},
		},
		{
			Key: "docker.readiness_timeout_seconds",
			DefaultValue: func() string {
				return fmt.Sprintf("%d", config.ReadinessTimeoutSeconds())
			},
		},
	}

	for _, e := range engines.All() {
		eng := e // capture for closure
		settings = append(settings, config.Setting{
			Key:          "docker.engines." + eng.Name() + ".default_image",
			DefaultValue: eng.DefaultImage,
		})
	}

	return settings
}

func NewConfigCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "View and edit forge docker configuration",
	}

	root.AddCommand(newConfigShowCommand())
	root.AddCommand(newConfigSetCommand())

	return root
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current docker configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows := config.Snapshot(dockerSettings())
			tableRows := make([][]string, len(rows))
			for i, r := range rows {
				tableRows[i] = []string{r.Key, r.Value, string(r.Source)}
			}
			ui.RenderTable([]string{"KEY", "VALUE", "SOURCE"}, tableRows)
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a docker configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawKey, value := args[0], args[1]

			// Normalise: prepend "docker." if missing.
			key := rawKey
			if !strings.HasPrefix(key, "docker.") {
				key = "docker." + key
			}

			if _, err := config.ValidateKey("docker", dockerSettings(), key); err != nil {
				logger.Error(err.Error())
				return err
			}

			if err := config.Set(key, value); err != nil {
				logger.Error(fmt.Sprintf("failed to save config: %s", err.Error()))
				return err
			}

			logger.Success(fmt.Sprintf("Set %s = %s", key, value))
			return nil
		},
	}
}
