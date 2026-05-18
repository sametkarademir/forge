package commands

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
)

const customImageLabel = "Enter custom..."

// promptEngine shows a selection of available engines rendered as "name (default image)".
// Returns the engine name (not the display string).
func promptEngine() (string, error) {
	all := engines.All()
	displayOpts := make([]string, len(all))
	nameByDisplay := make(map[string]string, len(all))
	for i, e := range all {
		display := fmt.Sprintf("%s  (%s)", e.Name(), e.DefaultImage())
		displayOpts[i] = display
		nameByDisplay[display] = e.Name()
	}
	chosen, err := ui.Select("Database engine", displayOpts, displayOpts[0])
	if err != nil {
		return "", err
	}
	return nameByDisplay[chosen], nil
}

// promptImage shows a Select with the engine's default + locally-available images,
// plus an "Enter custom..." fallback. Uses service.ListLocalImages to avoid
// importing the Docker client directly in the commands layer.
func promptImage(ctx context.Context, engineName string) (string, error) {
	eng, ok := engines.Get(engineName)
	if !ok {
		return "", engines.ErrUnknownEngine(engineName)
	}

	defaultImage := config.EngineDefaultImage(engineName)
	if defaultImage == "" {
		defaultImage = eng.DefaultImage()
	}

	installed, _ := service.ListLocalImages(ctx, eng)

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

// promptVhost asks for a RabbitMQ virtual host. Defaults to "/" (the RabbitMQ default).
func promptVhost() (string, error) {
	return ui.Text("Virtual host", "/", func(s string) error {
		if s == "" {
			return fmt.Errorf("virtual host cannot be empty")
		}
		return nil
	})
}

func promptUser() (string, error) {
	return ui.Text("DB username", config.DefaultUser(), func(s string) error {
		if s == "" {
			return fmt.Errorf("user cannot be empty")
		}
		return nil
	})
}

// generatePassword produces a 20-character password that satisfies the mssql
// complexity policy (and therefore any less-strict engine as well):
// uppercase, lowercase, digit, one special character, and random padding.
func generatePassword() string {
	const (
		upper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lower   = "abcdefghjkmnpqrstuvwxyz"
		digits  = "23456789"
		special = "!@#$%^&*"
		all     = upper + lower + digits + special
	)
	buf := make([]byte, 20)
	// Guarantee at least one of each required class at known positions.
	buf[0] = pickRand(upper)
	buf[1] = pickRand(lower)
	buf[2] = pickRand(digits)
	buf[3] = pickRand(special)
	for i := 4; i < len(buf); i++ {
		buf[i] = pickRand(all)
	}
	// Shuffle the fixed positions to avoid predictable prefix.
	for i := len(buf) - 1; i > 0; i-- {
		j := int(pickRand(all)) % (i + 1)
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func pickRand(charset string) byte {
	n := len(charset)
	b := make([]byte, 1)
	for {
		_, _ = rand.Read(b)
		if int(b[0]) < (256/n)*n { // avoid modulo bias
			return charset[int(b[0])%n]
		}
	}
}

func promptPassword(eng engines.Engine) (string, error) {
	gen, err := ui.Confirm("Generate a secure password?")
	if err != nil {
		return "", err
	}
	if gen {
		pwd := generatePassword()
		// Retry if engine rejects (shouldn't happen for well-formed generator).
		for eng.ValidatePassword(pwd) != nil {
			pwd = generatePassword()
		}
		fmt.Printf("  Generated password: %s\n", pwd)
		fmt.Println("  (Save this — it won't be shown again.)")
		return pwd, nil
	}
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

// promptHostPort asks for an optional host-side port.
// Returns 0 when the user leaves the field empty (auto-allocate at run time).
func promptHostPort(rangeStart, rangeEnd int) (int, error) {
	raw, err := ui.Text(
		fmt.Sprintf("Host port  (leave empty to auto-assign from %d–%d)", rangeStart, rangeEnd),
		"",
		func(s string) error {
			if s == "" {
				return nil
			}
			n, parseErr := strconv.Atoi(s)
			if parseErr != nil || n < 1 || n > 65535 {
				return fmt.Errorf("must be a number between 1 and 65535, or leave empty for auto")
			}
			return nil
		},
	)
	if err != nil {
		return 0, err
	}
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
