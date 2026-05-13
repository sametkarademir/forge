package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewWizardCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "wizard",
		Short: "Interactively create a managed database container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, confirmed, err := runWizard(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			if !confirmed {
				logger.Info("Aborted.")
				return nil
			}
			_, err = service.CreateProject(cmd.Context(), opts)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}
}

func runWizard(ctx context.Context) (service.CreateOptions, bool, error) {
	logger.Info("forge docker create wizard — answer each prompt (Enter = accept default)\n")

	// 1. Project name
	projectName, err := ui.Text("Project name", "", service.ValidateProjectName)
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	// 2. Engine
	allEngines := engines.All()
	engineNames := make([]string, len(allEngines))
	for i, e := range allEngines {
		engineNames[i] = e.Name()
	}
	engineName, err := ui.Select("Database engine", engineNames, engineNames[0])
	if err != nil {
		return service.CreateOptions{}, false, err
	}
	eng, _ := engines.Get(engineName)

	// 3. Image — shows locally-installed images for this engine first.
	image, err := promptImage(ctx, engineName)
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	// 4. User
	user, err := ui.Text("Database user", config.DefaultUser(), func(s string) error {
		if s == "" {
			return fmt.Errorf("user cannot be empty")
		}
		return nil
	})
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	// 5. Password (no default — re-prompt on engine rule violations)
	password, err := promptPassword(eng)
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	// 6. Database name
	database, err := ui.Text("Database name", config.DefaultDB(), func(s string) error {
		if s == "" {
			return fmt.Errorf("database name cannot be empty")
		}
		return nil
	})
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	// 7. Port — suggest the next auto-allocated port as default
	suggestedPort, _ := service.NextFreePort(config.PortRangeStart(), config.PortRangeEnd())
	if suggestedPort == 0 {
		suggestedPort = config.PortRangeStart()
	}
	portStr, err := ui.Text("Host port", strconv.Itoa(suggestedPort), func(s string) error {
		p, convErr := strconv.Atoi(s)
		if convErr != nil {
			return fmt.Errorf("must be a number")
		}
		if p < 1024 || p > 65535 {
			return fmt.Errorf("must be between 1024 and 65535")
		}
		if !service.IsPortFree(p) {
			return fmt.Errorf("port %d is occupied by another process", p)
		}
		return nil
	})
	if err != nil {
		return service.CreateOptions{}, false, err
	}
	port, _ := strconv.Atoi(portStr)

	// Summary
	fmt.Println()
	ui.RenderTable([]string{"Setting", "Value"}, [][]string{
		{"Project", projectName},
		{"Engine", engineName},
		{"Image", image},
		{"User", user},
		{"Password", "****"},
		{"Database", database},
		{"Host port", portStr},
	})
	fmt.Println()

	confirmed, err := ui.Confirm("Proceed with creating this project?")
	if err != nil {
		return service.CreateOptions{}, false, err
	}

	return service.CreateOptions{
		ProjectName: projectName,
		Engine:      engineName,
		Image:       image,
		User:        user,
		Password:    password,
		Database:    database,
		HostPort:    port,
	}, confirmed, nil
}
