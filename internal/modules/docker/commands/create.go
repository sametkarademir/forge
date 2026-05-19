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

// NewCreateCommand returns the preset-creation command.
// When all required flags are provided (--engine, --user, --password, --db) it runs
// non-interactively. Otherwise it falls through to the interactive wizard.
func NewCreateCommand() *cobra.Command {
	var (
		flagEngine   string
		flagUser     string
		flagPassword string
		flagDB       string
		flagPort     int
		flagImage    string
		flagOptions  []string
		flagRun      bool
	)

	cmd := &cobra.Command{
		Use:   "create [<preset-name>]",
		Short: "Create a database preset (interactive wizard or --flags)",
		Long: `Create a preset interactively, or supply all required flags for non-interactive use:

  forge docker create mydb --engine postgres --user alice --password S3cret! --db myapp
  forge docker create rabbit --engine rabbitmq --user admin --password P@ssw0rd --db / --option mgmt_host_port=15672`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Flag-driven mode: engine + user + password + db all provided.
			flagsProvided := flagEngine != "" && flagUser != "" && flagPassword != "" && flagDB != ""
			if flagsProvided {
				name := ""
				if len(args) > 0 {
					name = args[0]
				}
				return runCreateFromFlags(cmd.Context(), name, flagEngine, flagUser, flagPassword, flagDB, flagPort, flagImage, flagOptions, flagRun)
			}

			if !ui.IsInteractive() {
				return fmt.Errorf(
					"forge docker create requires an interactive terminal, or use flags:\n" +
						"  forge docker create <name> --engine <engine> --user <user> --password <pass> --db <db>",
				)
			}
			return runCreateWizard(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&flagEngine, "engine", "", "Engine name (postgres, mysql, mssql, redis, rabbitmq)")
	cmd.Flags().StringVar(&flagUser, "user", "", "DB username")
	cmd.Flags().StringVar(&flagPassword, "password", "", "DB password")
	cmd.Flags().StringVar(&flagDB, "db", "", "Database name (or vhost for RabbitMQ, index for Redis)")
	cmd.Flags().IntVar(&flagPort, "port", 0, "Host port (0 = auto-assign)")
	cmd.Flags().StringVar(&flagImage, "image", "", "Docker image (default: engine's default)")
	cmd.Flags().StringArrayVar(&flagOptions, "option", nil, "Engine-specific option as key=value (repeatable)")
	cmd.Flags().BoolVar(&flagRun, "run", false, "Start the container immediately after creating the preset")
	return cmd
}

func runCreateFromFlags(ctx context.Context, name, engineName, user, password, db string, port int, image string, rawOpts []string, runNow bool) error {
	eng, ok := engines.Get(engineName)
	if !ok {
		return engines.ErrUnknownEngine(engineName)
	}

	if name == "" {
		return fmt.Errorf("preset name required as positional argument — usage: forge docker create <name> --engine <engine>")
	}
	if err := preset.ValidateName(name); err != nil {
		return err
	}
	if preset.Exists(name) {
		return fmt.Errorf("preset %q already exists — choose another name", name)
	}
	if err := eng.ValidatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	if image == "" {
		cfg := config.EngineDefaultImage(engineName)
		if cfg != "" {
			image = cfg
		} else {
			image = eng.DefaultImage()
		}
	}

	options := map[string]string{}
	for _, kv := range rawOpts {
		idx := -1
		for i, ch := range kv {
			if ch == '=' {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("invalid --option %q: must be key=value", kv)
		}
		options[kv[:idx]] = kv[idx+1:]
	}
	if len(options) == 0 {
		options = nil
	}

	p := &preset.Preset{
		SchemaVersion: 2,
		Name:          name,
		Engine:        engineName,
		Image:         image,
		Database:      db,
		Username:      user,
		Password:      password,
		InternalPort:  eng.DefaultPort(),
		HostPort:      port,
		Options:       options,
		CreatedAt:     time.Now().UTC(),
	}
	if err := service.CreatePreset(ctx, p, true); err != nil {
		logger.Error(err.Error())
		return err
	}
	logger.Success(fmt.Sprintf("Preset %q saved to ~/.forge/presets/%s.yaml", name, name))

	if !runNow {
		logger.Info(fmt.Sprintf("Run with: forge docker run %s", name))
		return nil
	}

	info, err := service.RunPreset(ctx, name, service.RunOptions{})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if info.ConnectionString != "" {
		logger.Info("  Connection: " + info.ConnectionString)
	}
	for _, ep := range info.Endpoints {
		logger.Info("  " + ep.Label + ": " + ep.Value)
	}
	return nil
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
	var db string
	switch engineName {
	case "redis":
		db = "0"
	case "rabbitmq":
		db, err = promptVhost()
		if err != nil {
			return err
		}
	default:
		db, err = promptDB()
		if err != nil {
			return err
		}
	}

	// Step 5: Host port (optional)
	hostPort, err := promptHostPort(config.PortRangeStart(), config.PortRangeEnd())
	if err != nil {
		return err
	}

	// Step 5b: Engine-specific extra options (e.g. RabbitMQ Management UI port)
	var options map[string]string
	if wp, ok := eng.(engines.WizardPromptProvider); ok {
		for _, op := range wp.WizardPrompts(image) {
			validate := op.Validate
			val, err := ui.Text(op.Label, op.Default, validate)
			if err != nil {
				return err
			}
			if val != "" {
				if options == nil {
					options = make(map[string]string)
				}
				options[op.Key] = val
			}
		}
	}

	// Step 6: Confirmation summary
	hostPortDisplay := fmt.Sprintf("auto (%d–%d)", config.PortRangeStart(), config.PortRangeEnd())
	if hostPort != 0 {
		hostPortDisplay = strconv.Itoa(hostPort)
	}
	dbLabel := "Database"
	if engineName == "rabbitmq" {
		dbLabel = "Virtual host"
	}
	summaryRows := [][]string{
		{"Preset name", presetName},
		{"Engine", engineName},
		{"Image", image},
		{"Username", user},
		{"Password", "****"},
		{dbLabel, db},
		{"Host port", hostPortDisplay},
	}
	for k, v := range options {
		if v != "" {
			summaryRows = append(summaryRows, []string{k, v})
		}
	}
	logger.Info("")
	ui.RenderTable([]string{"Setting", "Value"}, summaryRows)
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
		SchemaVersion: 2,
		Name:          presetName,
		Engine:        engineName,
		Image:         image,
		Database:      db,
		Username:      user,
		Password:      password,
		InternalPort:  eng.DefaultPort(),
		HostPort:      hostPort,
		Options:       options,
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
	for _, ep := range info.Endpoints {
		logger.Info("  " + ep.Label + ": " + ep.Value)
	}
	return nil
}
