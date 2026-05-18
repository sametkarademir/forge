package client

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ListByLabels returns all containers that carry every label in the provided map.
func (dc *DockerClient) ListByLabels(ctx context.Context, labels map[string]string) ([]types.Container, error) {
	f := filters.NewArgs()
	for k, v := range labels {
		f.Add("label", k+"="+v)
	}
	return dc.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
}

// ListManaged returns all containers carrying the forge.managed=true label.
func (dc *DockerClient) ListManaged(ctx context.Context) ([]types.Container, error) {
	return dc.ListByLabels(ctx, map[string]string{"forge.managed": "true"})
}

// InspectByLabels finds and inspects the first container matching all provided labels.
func (dc *DockerClient) InspectByLabels(ctx context.Context, labels map[string]string) (types.ContainerJSON, error) {
	containers, err := dc.ListByLabels(ctx, labels)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	if len(containers) == 0 {
		return types.ContainerJSON{}, fmt.Errorf("no managed container found for labels %v", labels)
	}
	return dc.cli.ContainerInspect(ctx, containers[0].ID)
}

// InspectByProject finds and inspects the managed container for a project.
// Kept as a thin adapter over InspectByLabels for backward compatibility.
func (dc *DockerClient) InspectByProject(ctx context.Context, project string) (types.ContainerJSON, error) {
	c, err := dc.InspectByLabels(ctx, map[string]string{
		"forge.managed": "true",
		"forge.project": project,
	})
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("no managed container found for project %q", project)
	}
	return c, nil
}

// InspectByPreset finds and inspects the managed container for a preset.
func (dc *DockerClient) InspectByPreset(ctx context.Context, preset string) (types.ContainerJSON, error) {
	c, err := dc.InspectByLabels(ctx, map[string]string{
		"forge.managed": "true",
		"forge.preset":  preset,
	})
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("no managed container found for preset %q", preset)
	}
	return c, nil
}

// Inspect returns the full container details for the given ID or name.
func (dc *DockerClient) Inspect(ctx context.Context, idOrName string) (types.ContainerJSON, error) {
	return dc.cli.ContainerInspect(ctx, idOrName)
}

// RunContainer creates and starts a container from RunConfig.
// It attempts a best-effort image pull before creating the container;
// use PullImage for explicit pre-pull with error surfacing.
func (dc *DockerClient) RunContainer(ctx context.Context, cfg RunConfig) (string, error) {
	// Best-effort pull — already-local images are a no-op.
	if r, err := dc.cli.ImagePull(ctx, cfg.Image, pullOpts()); err == nil {
		_, _ = io.Copy(io.Discard, r)
		_ = r.Close()
	}

	envSlice := make([]string, 0, len(cfg.EnvVars))
	for k, v := range cfg.EnvVars {
		envSlice = append(envSlice, k+"="+v)
	}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", cfg.ContainerPort))
	portBindings := nat.PortMap{
		containerPort: []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", cfg.HostPort)},
		},
	}
	exposedPorts := nat.PortSet{containerPort: struct{}{}}
	for _, ep := range cfg.ExtraPorts {
		p := nat.Port(fmt.Sprintf("%d/tcp", ep.ContainerPort))
		portBindings[p] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", ep.HostPort)}}
		exposedPorts[p] = struct{}{}
	}

	mounts := []mount.Mount{{
		Type:   mount.TypeVolume,
		Source: cfg.VolumeName,
		Target: cfg.VolumeTarget,
	}}

	containerConfig := &container.Config{
		Image:        cfg.Image,
		Env:          envSlice,
		Labels:       cfg.Labels,
		ExposedPorts: exposedPorts,
	}
	if len(cfg.Cmd) > 0 {
		containerConfig.Cmd = cfg.Cmd
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
	}
	networkConfig := &dockernetwork.NetworkingConfig{
		EndpointsConfig: map[string]*dockernetwork.EndpointSettings{
			cfg.NetworkName: {},
		},
	}

	resp, err := dc.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	if err := dc.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	return resp.ID, nil
}

// StartContainer starts an already-created (stopped) container.
func (dc *DockerClient) StartContainer(ctx context.Context, id string) error {
	return dc.cli.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a running container; ignores "not running" errors.
func (dc *DockerClient) StopContainer(ctx context.Context, id string) error {
	err := dc.cli.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil && !isNotRunningError(err) {
		return err
	}
	return nil
}

// RemoveContainer force-removes a container.
func (dc *DockerClient) RemoveContainer(ctx context.Context, id string) error {
	return dc.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
}

// LogsOptions controls ContainerLogs behaviour.
type LogsOptions struct {
	Follow bool
	Tail   string // number of lines or "all"; empty means all
	Since  string // duration string e.g. "5m", "1h", or RFC3339 timestamp
}

// ContainerLogs returns a stream of the container's stdout and stderr.
// The stream is Docker's multiplexed format; callers should use
// stdcopy.StdCopy(stdout, stderr, rc) to demultiplex correctly.
func (dc *DockerClient) ContainerLogs(ctx context.Context, id string, opts LogsOptions) (io.ReadCloser, error) {
	tail := opts.Tail
	if tail == "" {
		tail = "all"
	}
	return dc.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: false,
		Tail:       tail,
		Since:      opts.Since,
	})
}
