package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

// NewCreateCommand returns the interactive preset-creation wizard.
func NewCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a database preset (interactive wizard)",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("'create' takes no arguments — run it bare: forge docker create")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !ui.IsInteractive() {
				return fmt.Errorf("forge docker create requires an interactive terminal")
			}
			return runCreateWizard(cmd.Context())
		},
	}
}

func runCreateWizard(ctx context.Context) error {
	logger.Info("forge docker create — press Enter to accept defaults")
	logger.Info("")

	// Step 1: Engine
	engineName, err := promptEngine()
	if err != nil {
		return err
	}
	eng, _ := engines.Get(engineName)

	// Step 2: Preset name — validated and checked for collision
	presetName, err := ui.Text("Preset name", "", func(s string) error {
		if err := preset.ValidateName(s); err != nil {
			return err
		}
		if preset.Exists(s) {
			return fmt.Errorf("preset %q already exists — choose another name", s)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Step 3: Image
	image, err := promptImage(ctx, engineName)
	if err != nil {
		return err
	}

	// Step 4: Credentials and database name
	user, err := promptUser()
	if err != nil {
		return err
	}
	password, err := promptPassword(eng)
	if err != nil {
		return err
	}
	db, err := promptDB()
	if err != nil {
		return err
	}

	// Step 5: Host port (optional)
	hostPort, err := promptHostPort(config.PortRangeStart(), config.PortRangeEnd())
	if err != nil {
		return err
	}

	// Step 6: Confirmation summary
	hostPortDisplay := fmt.Sprintf("auto (%d–%d)", config.PortRangeStart(), config.PortRangeEnd())
	if hostPort != 0 {
		hostPortDisplay = strconv.Itoa(hostPort)
	}
	logger.Info("")
	ui.RenderTable([]string{"Setting", "Value"}, [][]string{
		{"Preset name", presetName},
		{"Engine", engineName},
		{"Image", image},
		{"Username", user},
		{"Password", "****"},
		{"Database", db},
		{"Host port", hostPortDisplay},
	})
	logger.Info("")

	ok, err := ui.Confirm("Save this preset?")
	if err != nil {
		return err
	}
	if !ok {
		logger.Info("Aborted.")
		return nil
	}

	// Steps 7–8: Save preset and pull image
	p := &preset.Preset{
		SchemaVersion: 1,
		Name:          presetName,
		Engine:        engineName,
		Image:         image,
		Database:      db,
		Username:      user,
		Password:      password,
		InternalPort:  eng.DefaultPort(),
		HostPort:      hostPort,
		CreatedAt:     time.Now().UTC(),
	}
	if err := service.CreatePreset(ctx, p, true); err != nil {
		logger.Error(err.Error())
		return err
	}
	logger.Success(fmt.Sprintf("Preset %q saved to ~/.forge/presets/%s.yaml", presetName, presetName))
	logger.Info("")

	// Step 9: Offer to run immediately
	runNow, err := ui.ConfirmDefault("Run now?", true)
	if err != nil {
		return err
	}
	if !runNow {
		logger.Info(fmt.Sprintf("Run later with: forge docker run %s", presetName))
		return nil
	}

	info, err := service.RunPreset(ctx, presetName, service.RunOptions{})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if info.ConnectionString != "" {
		logger.Info("  Connection: " + info.ConnectionString)
	}
	return nil
}
