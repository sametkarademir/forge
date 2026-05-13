package commands

import (
	"context"
	"fmt"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/ui"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
)

const customImageLabel = "Enter custom..."

func promptEngine() (string, error) {
	all := engines.All()
	names := make([]string, len(all))
	for i, e := range all {
		names[i] = e.Name()
	}
	return ui.Select("Database engine", names, names[0])
}

// promptImage shows a Select with the engine's default + locally-installed images,
// plus an "Enter custom..." option that falls back to free-text input.
func promptImage(ctx context.Context, engineName string) (string, error) {
	eng, ok := engines.Get(engineName)
	if !ok {
		return "", engines.ErrUnknownEngine(engineName)
	}

	// Effective default: config override first, then engine compiled default.
	defaultImage := config.EngineDefaultImage(engineName)
	if defaultImage == "" {
		defaultImage = eng.DefaultImage()
	}

	// Best-effort: list images from local daemon. Degrade gracefully on failure.
	var installed []dockerclient.ImageRef
	if dc, err := dockerclient.NewClient(); err == nil {
		installed, _ = dc.ListImages(ctx, eng.ImageRepos())
	}

	// Build display options and a parallel slice of actual values.
	displayOpts := []string{defaultImage + "  (default)"}
	values := []string{defaultImage}

	for _, ref := range installed {
		candidate := ref.String()
		if candidate == defaultImage {
			continue
		}
		displayOpts = append(displayOpts, candidate+"  (installed)")
		values = append(values, candidate)
	}
	displayOpts = append(displayOpts, customImageLabel)
	values = append(values, customImageLabel)

	chosen, err := ui.Select("Docker image", displayOpts, displayOpts[0])
	if err != nil {
		return "", fmt.Errorf("image selection cancelled: %w", err)
	}

	for i, opt := range displayOpts {
		if opt == chosen {
			if values[i] == customImageLabel {
				return ui.Text("Docker image", defaultImage, func(s string) error {
					if s == "" {
						return fmt.Errorf("image cannot be empty")
					}
					return nil
				})
			}
			return values[i], nil
		}
	}
	return defaultImage, nil
}

func promptUser() (string, error) {
	return ui.Text("DB username", config.DefaultUser(), func(s string) error {
		if s == "" {
			return fmt.Errorf("user cannot be empty")
		}
		return nil
	})
}

func promptPassword(eng engines.Engine) (string, error) {
	return ui.Password("DB password", eng.ValidatePassword)
}

func promptDB() (string, error) {
	return ui.Text("Database name", config.DefaultDB(), func(s string) error {
		if s == "" {
			return fmt.Errorf("database name cannot be empty")
		}
		return nil
	})
}
