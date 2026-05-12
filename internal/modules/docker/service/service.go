package service

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
)

var projectNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// ProjectInfo represents a forge-managed container's state.
type ProjectInfo struct {
	Name             string
	Engine           string
	ContainerID      string
	ContainerName    string
	VolumeName       string
	HostPort         int
	Status           string
	Image            string
	CreatedAt        time.Time
	Uptime           time.Duration
	EnvSummary       map[string]string
	ConnectionString string
}

// CreateOptions holds parameters for creating a new project container.
type CreateOptions struct {
	ProjectName string
	Engine      string
	Image       string
	User        string
	Password    string
	Database    string
}

// CreateProject validates, allocates, and starts a new database container for a project.
func CreateProject(ctx context.Context, opts CreateOptions) (*ProjectInfo, error) {
	if !projectNameRe.MatchString(opts.ProjectName) {
		return nil, fmt.Errorf("invalid project name %q — must match ^[a-z0-9][a-z0-9-]{0,62}$", opts.ProjectName)
	}

	eng, ok := engines.Get(opts.Engine)
	if !ok {
		return nil, engines.ErrUnknownEngine(opts.Engine)
	}

	if err := eng.ValidatePassword(opts.Password); err != nil {
		return nil, fmt.Errorf("password does not meet %s requirements: %s", opts.Engine, err.Error())
	}

	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}

	if _, err := dc.InspectByProject(ctx, opts.ProjectName); err == nil {
		return nil, fmt.Errorf(
			"project %q already exists — use 'forge docker reset %s' or 'forge docker remove %s'",
			opts.ProjectName, opts.ProjectName, opts.ProjectName,
		)
	}

	port, err := NextFreePort(config.PortRangeStart(), config.PortRangeEnd())
	if err != nil {
		return nil, fmt.Errorf(
			"port range %d–%d is exhausted — increase docker.port_range_end in ~/.forge/config.yaml",
			config.PortRangeStart(), config.PortRangeEnd(),
		)
	}

	if err := dc.EnsureNetwork(ctx, NetworkName()); err != nil {
		return nil, fmt.Errorf("failed to ensure network %s: %w", NetworkName(), err)
	}

	volumeName := VolumeName(opts.ProjectName, opts.Engine)
	now := time.Now().UTC()
	volumeLabels := map[string]string{
		"forge.managed":    "true",
		"forge.project":    opts.ProjectName,
		"forge.engine":     opts.Engine,
		"forge.created_at": now.Format(time.RFC3339),
	}
	if err := dc.VolumeCreate(ctx, volumeName, volumeLabels); err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	image := opts.Image
	if image == "" {
		image = eng.DefaultImage()
	}

	containerName := ContainerName(opts.ProjectName, opts.Engine)
	containerLabels := map[string]string{
		"forge.managed":    "true",
		"forge.project":    opts.ProjectName,
		"forge.engine":     opts.Engine,
		"forge.created_at": now.Format(time.RFC3339),
		"forge.host_port":  strconv.Itoa(port),
		"forge.user":       opts.User,
		"forge.db":         opts.Database,
	}

	_, err = dc.RunContainer(ctx, dockerclient.RunConfig{
		Name:          containerName,
		Image:         image,
		EnvVars:       eng.EnvVars(opts.User, opts.Password, opts.Database),
		HostPort:      port,
		ContainerPort: eng.DefaultPort(),
		VolumeTarget:  eng.DataDir(),
		VolumeName:    volumeName,
		Labels:        containerLabels,
		NetworkName:   NetworkName(),
	})
	if err != nil {
		return nil, err
	}

	connStr := eng.ConnectionString("localhost", port, opts.User, opts.Password, opts.Database)
	logger.Success(fmt.Sprintf("Created %s on port %d", containerName, port))
	logger.Info("  Connection: " + connStr)

	WaitForReady(port, config.ReadinessTimeoutSeconds())

	return &ProjectInfo{
		Name:             opts.ProjectName,
		Engine:           opts.Engine,
		ContainerName:    containerName,
		VolumeName:       volumeName,
		HostPort:         port,
		Status:           "running",
		Image:            image,
		CreatedAt:        now,
		ConnectionString: connStr,
	}, nil
}

// WaitForReady polls localhost:<port> via TCP until the DB accepts connections or timeout.
func WaitForReady(port, timeoutSecs int) {
	addr := fmt.Sprintf("localhost:%d", port)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	start := time.Now()
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			fmt.Printf("\r✓ DB ready (in %s)                        \n", time.Since(start).Round(time.Second))
			return
		}
		fmt.Printf("\r⠿ waiting for DB…")
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println()
	logger.Warn(fmt.Sprintf(
		"DB did not become ready within %ds — the connection string is still valid; try again in a moment.",
		timeoutSecs,
	))
}

// ListProjects returns a ProjectInfo for every forge-managed container.
func ListProjects(ctx context.Context) ([]*ProjectInfo, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}
	containers, err := dc.ListManaged(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]*ProjectInfo, 0, len(containers))
	for _, c := range containers {
		infos = append(infos, listContainerToInfo(c))
	}
	return infos, nil
}

func listContainerToInfo(c dockertypes.Container) *ProjectInfo {
	port, _ := strconv.Atoi(c.Labels["forge.host_port"])
	createdAt, _ := time.Parse(time.RFC3339, c.Labels["forge.created_at"])

	var uptime time.Duration
	if c.State == "running" {
		uptime = time.Since(time.Unix(c.Created, 0))
	}

	id := c.ID
	if len(id) > 12 {
		id = id[:12]
	}

	return &ProjectInfo{
		Name:          c.Labels["forge.project"],
		Engine:        c.Labels["forge.engine"],
		ContainerID:   id,
		ContainerName: ContainerName(c.Labels["forge.project"], c.Labels["forge.engine"]),
		VolumeName:    VolumeName(c.Labels["forge.project"], c.Labels["forge.engine"]),
		HostPort:      port,
		Status:        c.State,
		Image:         c.Image,
		CreatedAt:     createdAt,
		Uptime:        uptime,
	}
}

// GetProjectStatus returns detailed info for a single managed container.
func GetProjectStatus(ctx context.Context, project string) (*ProjectInfo, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}

	inspected, err := dc.InspectByProject(ctx, project)
	if err != nil {
		known := knownProjectNames(ctx, dc)
		if len(known) > 0 {
			return nil, fmt.Errorf("no managed container found for project %q\n  Known projects: %s",
				project, strings.Join(known, ", "))
		}
		return nil, fmt.Errorf("no managed container found for project %q", project)
	}

	return buildProjectInfo(inspected, true), nil
}

// GetConnectionString returns the real (unmasked) DSN for a project's container.
func GetConnectionString(ctx context.Context, project string) (string, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return "", err
	}

	inspected, err := dc.InspectByProject(ctx, project)
	if err != nil {
		return "", fmt.Errorf("no managed container found for project %q", project)
	}

	return realConnectionString(inspected), nil
}

// ResetProject stops, wipes, and recreates a project's container on the same port.
func ResetProject(ctx context.Context, project string) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}

	inspected, err := dc.InspectByProject(ctx, project)
	if err != nil {
		return fmt.Errorf("no managed container found for project %q", project)
	}

	port, _ := strconv.Atoi(inspected.Config.Labels["forge.host_port"])
	if !IsPortFree(port) {
		return fmt.Errorf(
			"port %d is occupied by another process\n  Free the port or use:\n    forge docker remove %s && forge docker create %s --engine %s",
			port, project, project, inspected.Config.Labels["forge.engine"],
		)
	}

	engineName := inspected.Config.Labels["forge.engine"]
	user := inspected.Config.Labels["forge.user"]
	db := inspected.Config.Labels["forge.db"]
	password := envValue(inspected.Config.Env, passwordEnvKey(engineName))
	image := inspected.Config.Image

	_ = dc.StopContainer(ctx, inspected.ID)
	if err := dc.RemoveContainer(ctx, inspected.ID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	if err := dc.VolumeRemove(ctx, VolumeName(project, engineName)); err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	_, err = CreateProject(ctx, CreateOptions{
		ProjectName: project,
		Engine:      engineName,
		Image:       image,
		User:        user,
		Password:    password,
		Database:    db,
	})
	return err
}

// RemoveProject stops and removes a project's container, volume, and network membership.
func RemoveProject(ctx context.Context, project string) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}

	inspected, err := dc.InspectByProject(ctx, project)
	if err != nil {
		// Check for an unmanaged container with a matching name prefix.
		candidates, _ := dc.FindByNamePrefix(ctx, project)
		for _, c := range candidates {
			if c.Labels["forge.managed"] != "true" {
				return fmt.Errorf(
					"container %q exists but is not managed by forge\n  (missing label forge.managed=true) — refusing to remove",
					c.Names[0],
				)
			}
		}
		// Truly not found — idempotent success.
		return nil
	}

	engineName := inspected.Config.Labels["forge.engine"]
	_ = dc.StopContainer(ctx, inspected.ID)

	if err := dc.RemoveContainer(ctx, inspected.ID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	_ = dc.VolumeRemove(ctx, VolumeName(project, engineName))
	dc.NetworkDisconnect(ctx, NetworkName(), inspected.ID)
	return nil
}

// --- helpers ---

func buildProjectInfo(c dockertypes.ContainerJSON, maskPassword bool) *ProjectInfo {
	labels := c.Config.Labels
	port, _ := strconv.Atoi(labels["forge.host_port"])
	createdAt, _ := time.Parse(time.RFC3339, labels["forge.created_at"])
	engineName := labels["forge.engine"]
	project := labels["forge.project"]

	var uptime time.Duration
	if c.State.Running {
		if t, err := time.Parse(time.RFC3339Nano, c.State.StartedAt); err == nil {
			uptime = time.Since(t)
		}
	}

	// Env summary with optional password masking
	envSummary := make(map[string]string)
	pwKey := passwordEnvKey(engineName)
	for _, kv := range c.Config.Env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		if maskPassword && k == pwKey {
			envSummary[k] = "****"
		} else {
			envSummary[k] = v
		}
	}

	connStr := realConnectionString(c)
	if maskPassword {
		connStr = maskPasswordInDSN(connStr)
	}

	id := c.ID
	if len(id) > 12 {
		id = id[:12]
	}

	return &ProjectInfo{
		Name:             project,
		Engine:           engineName,
		ContainerID:      id,
		ContainerName:    ContainerName(project, engineName),
		VolumeName:       VolumeName(project, engineName),
		HostPort:         port,
		Status:           c.State.Status,
		Image:            c.Config.Image,
		CreatedAt:        createdAt,
		Uptime:           uptime,
		EnvSummary:       envSummary,
		ConnectionString: connStr,
	}
}

func realConnectionString(c dockertypes.ContainerJSON) string {
	labels := c.Config.Labels
	engineName := labels["forge.engine"]
	port, _ := strconv.Atoi(labels["forge.host_port"])
	user := labels["forge.user"]
	db := labels["forge.db"]
	password := envValue(c.Config.Env, passwordEnvKey(engineName))

	eng, ok := engines.Get(engineName)
	if !ok {
		return ""
	}
	return eng.ConnectionString("localhost", port, user, password, db)
}

func passwordEnvKey(engineName string) string {
	eng, ok := engines.Get(engineName)
	if !ok {
		return ""
	}
	return eng.PasswordEnvKey()
}

func envValue(envSlice []string, key string) string {
	for _, kv := range envSlice {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1]
		}
	}
	return ""
}

func maskPasswordInDSN(dsn string) string {
	// postgres://user:pass@host:port/db
	if idx := strings.Index(dsn, "://"); idx != -1 {
		rest := dsn[idx+3:]
		atIdx := strings.LastIndex(rest, "@")
		if atIdx != -1 {
			userPass := rest[:atIdx]
			colonIdx := strings.Index(userPass, ":")
			if colonIdx != -1 {
				masked := dsn[:idx+3] + userPass[:colonIdx+1] + "****" + "@" + rest[atIdx+1:]
				return masked
			}
		}
	}
	// mssql: Server=…;Password=val;
	if strings.HasPrefix(dsn, "Server=") {
		parts := strings.Split(dsn, ";")
		for i, p := range parts {
			if strings.HasPrefix(strings.ToLower(p), "password=") {
				parts[i] = "Password=****"
			}
		}
		return strings.Join(parts, ";")
	}
	return dsn
}

func knownProjectNames(ctx context.Context, dc *dockerclient.DockerClient) []string {
	containers, err := dc.ListManaged(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(containers))
	for _, c := range containers {
		if n := c.Labels["forge.project"]; n != "" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}
