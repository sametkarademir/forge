package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
)

// RunOptions controls RunPreset behavior.
type RunOptions struct {
	NoWait   bool
	Timeout  int // seconds; 0 = config default
	HostPort int // 0 = auto-allocate; non-zero forces a specific port (used by ResetPreset)
}

// ContainerInfo is a snapshot of a preset container's runtime state.
type ContainerInfo struct {
	PresetName       string
	Engine           string
	ContainerID      string
	ContainerName    string
	VolumeName       string
	HostPort         int
	Status           string
	Image            string
	CreatedAt        time.Time
	ConnectionString string
	Endpoints        []engines.Endpoint
}

// PresetView is returned by ShowPreset.
type PresetView struct {
	Preset    *preset.Preset
	Status    string
	HostPort  int
	Primary   string             // password-masked primary connection string; empty if not running
	Endpoints []engines.Endpoint // additional endpoints; nil for DB engines
}

// RemoveMode controls which resources RemovePreset deletes.
type RemoveMode int

const (
	RemoveContainerVolume RemoveMode = iota // container + volume; keep preset YAML
	RemoveAll                               // container + volume + preset YAML
)

// Row is one line in the output of ListAll.
type Row struct {
	Name      string
	Engine    string
	Image     string
	HostPort  int
	Status    string // running | stopped | not created | orphaned | invalid | legacy
	CreatedAt time.Time
}

// PortConflictError is returned by ResetPreset when the preset's original port is occupied.
type PortConflictError struct {
	Port int
}

func (e *PortConflictError) Error() string {
	return fmt.Sprintf("port %d is occupied by another process", e.Port)
}

// CreatePreset saves a preset to disk and optionally pulls its image.
// If the pull fails the preset YAML is removed to avoid a half-saved state.
func CreatePreset(ctx context.Context, p *preset.Preset, pullImage bool) error {
	if err := preset.Save(p); err != nil {
		return fmt.Errorf("save preset: %w", err)
	}
	if !pullImage {
		return nil
	}
	dc, err := dockerclient.NewClient()
	if err != nil {
		_ = preset.Delete(p.Name)
		return err
	}
	exists, err := dc.ImageExists(ctx, p.Image)
	if err != nil {
		return fmt.Errorf("check image: %w", err)
	}
	if exists {
		return nil
	}
	logger.Info(fmt.Sprintf("Pulling %s…", p.Image))
	if err := dc.PullImage(ctx, p.Image); err != nil {
		return fmt.Errorf("pull image %s: %w", p.Image, err)
	}
	logger.Success(fmt.Sprintf("Pulled %s", p.Image))
	return nil
}

// ListLocalImages returns locally-available images for the given engine.
// Called by the create wizard to avoid importing the Docker client in commands.
func ListLocalImages(ctx context.Context, eng engines.Engine) ([]dockerclient.ImageRef, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}
	return dc.ListImages(ctx, eng.ImageRepos())
}

// RunPreset is idempotent: creates the container from the preset if missing,
// starts it if stopped, and no-ops if already running.
func RunPreset(ctx context.Context, name string, opts RunOptions) (*ContainerInfo, error) {
	p, err := preset.Load(name)
	if err != nil {
		return nil, err
	}

	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}

	// Container already exists?
	existing, inspectErr := dc.InspectByPreset(ctx, name)
	if inspectErr == nil {
		if existing.State.Running {
			return buildContainerInfo(&existing, p), nil
		}
		// Stopped — start it.
		if err := dc.StartContainer(ctx, existing.ID); err != nil {
			return nil, fmt.Errorf("start container: %w", err)
		}
		port, _ := strconv.Atoi(existing.Config.Labels["forge.host_port"])
		if !opts.NoWait {
			waitForPresetReady(port, readinessTimeout(opts.Timeout))
		}
		info := buildContainerInfo(&existing, p)
		info.Status = "running"
		return info, nil
	}

	// No container — create one.
	eng, ok := engines.Get(p.Engine)
	if !ok {
		return nil, engines.ErrUnknownEngine(p.Engine)
	}

	port := opts.HostPort
	if port == 0 {
		port = p.HostPort // preset's preferred port (0 = auto)
	}
	if port == 0 {
		port, err = NextFreePort(config.PortRangeStart(), config.PortRangeEnd())
		if err != nil {
			return nil, fmt.Errorf(
				"port range %d–%d exhausted — increase docker.port_range_end in ~/.forge/config.yaml",
				config.PortRangeStart(), config.PortRangeEnd(),
			)
		}
	} else if !IsPortFree(port) {
		return nil, &PortConflictError{Port: port}
	}

	if err := dc.EnsureNetwork(ctx, NetworkName()); err != nil {
		return nil, fmt.Errorf("ensure network: %w", err)
	}

	volName := PresetVolumeName(name)
	now := time.Now().UTC()
	if err := dc.VolumeCreate(ctx, volName, map[string]string{
		"forge.managed":    "true",
		"forge.preset":     name,
		"forge.engine":     p.Engine,
		"forge.created_at": now.Format(time.RFC3339),
	}); err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	imgTag := p.Image
	if imgTag == "" {
		imgTag = eng.DefaultImage()
	}

	containerName := PresetContainerName(name)
	if _, err := dc.RunContainer(ctx, dockerclient.RunConfig{
		Name:          containerName,
		Image:         imgTag,
		EnvVars:       eng.EnvVars(p.Username, p.Password, p.Database),
		HostPort:      port,
		ContainerPort: eng.DefaultPort(),
		VolumeTarget:  eng.DataDir(imgTag),
		VolumeName:    volName,
		Labels: map[string]string{
			"forge.managed":        "true",
			"forge.preset":         name,
			"forge.engine":         p.Engine,
			"forge.created_at":     now.Format(time.RFC3339),
			"forge.host_port":      strconv.Itoa(port),
			"forge.schema_version": "1",
			"forge.user":           p.Username,
			"forge.db":             p.Database,
		},
		NetworkName: NetworkName(),
	}); err != nil {
		return nil, err
	}

	logger.Success(fmt.Sprintf("Started %s on port %d", containerName, port))
	if !opts.NoWait {
		waitForPresetReady(port, readinessTimeout(opts.Timeout))
	}

	ci := eng.ConnectionInfo(engines.ConnArgs{
		Host:     "localhost",
		HostPort: port,
		User:     p.Username,
		Password: p.Password,
		Database: p.Database,
		Options:  p.Options,
	})
	return &ContainerInfo{
		PresetName:       name,
		Engine:           p.Engine,
		ContainerName:    containerName,
		VolumeName:       volName,
		HostPort:         port,
		Status:           "running",
		Image:            imgTag,
		CreatedAt:        now,
		ConnectionString: ci.Primary,
		Endpoints:        ci.Endpoints,
	}, nil
}

// StopPreset stops the container for a preset. Idempotent.
func StopPreset(ctx context.Context, name string) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return nil // not found — idempotent success
	}
	if err := dc.StopContainer(ctx, c.ID); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	logger.Success("Stopped " + PresetContainerName(name))
	return nil
}

// ResetPreset stops and wipes the container and volume, then recreates on the
// same port. Returns *PortConflictError if the original port is now occupied.
func ResetPreset(ctx context.Context, name string) error {
	if _, err := preset.Load(name); err != nil {
		return err
	}

	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}

	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return fmt.Errorf("no container found for preset %q — use 'forge docker run %s'", name, name)
	}

	port, _ := strconv.Atoi(c.Config.Labels["forge.host_port"])

	_ = dc.StopContainer(ctx, c.ID)
	if err := dc.RemoveContainer(ctx, c.ID); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	if err := dc.VolumeRemove(ctx, PresetVolumeName(name)); err != nil {
		return fmt.Errorf("remove volume: %w", err)
	}

	if port != 0 && !IsPortFree(port) {
		return &PortConflictError{Port: port}
	}

	_, err = RunPreset(ctx, name, RunOptions{HostPort: port})
	return err
}
