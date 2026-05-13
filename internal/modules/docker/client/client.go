package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sametkarademir/forge/internal/core/logger"
)

// DockerClient wraps the Docker SDK client.
type DockerClient struct {
	cli *dockerclient.Client
}

// RunConfig holds parameters for creating and starting a container.
type RunConfig struct {
	Name          string
	Image         string
	EnvVars       map[string]string
	HostPort      int
	ContainerPort int
	VolumeTarget  string // mount path inside container
	VolumeName    string
	Labels        map[string]string
	NetworkName   string
}

// NewClient creates a Docker client and pings the daemon.
func NewClient() (*DockerClient, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	dc := &DockerClient{cli: cli}
	if _, err := cli.Ping(context.Background()); err != nil {
		logger.Error("Docker is not running — run: open -a Docker")
		return nil, fmt.Errorf("docker daemon unreachable: %w", err)
	}
	return dc, nil
}

// ListManaged returns all containers carrying the forge.managed=true label.
func (dc *DockerClient) ListManaged(ctx context.Context) ([]types.Container, error) {
	f := filters.NewArgs(filters.KeyValuePair{Key: "label", Value: "forge.managed=true"})
	return dc.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
}

// InspectByProject finds and inspects the managed container for a project.
func (dc *DockerClient) InspectByProject(ctx context.Context, project string) (types.ContainerJSON, error) {
	f := filters.NewArgs(
		filters.KeyValuePair{Key: "label", Value: "forge.managed=true"},
		filters.KeyValuePair{Key: "label", Value: "forge.project=" + project},
	)
	containers, err := dc.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return types.ContainerJSON{}, err
	}
	if len(containers) == 0 {
		return types.ContainerJSON{}, fmt.Errorf("no managed container found for project %q", project)
	}
	return dc.cli.ContainerInspect(ctx, containers[0].ID)
}

// FindByNamePrefix returns containers whose name starts with "forge-<project>-".
func (dc *DockerClient) FindByNamePrefix(ctx context.Context, project string) ([]types.Container, error) {
	f := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "forge-" + project + "-"})
	return dc.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
}

// RunContainer creates and starts a container from RunConfig.
func (dc *DockerClient) RunContainer(ctx context.Context, cfg RunConfig) (string, error) {
	// Pull image (best-effort; continue if already local)
	reader, err := dc.cli.ImagePull(ctx, cfg.Image, image.PullOptions{})
	if err == nil {
		_, _ = io.Copy(io.Discard, reader)
		_ = reader.Close()
	}

	// Env slice
	envSlice := make([]string, 0, len(cfg.EnvVars))
	for k, v := range cfg.EnvVars {
		envSlice = append(envSlice, k+"="+v)
	}

	// Port bindings
	containerPort := nat.Port(fmt.Sprintf("%d/tcp", cfg.ContainerPort))
	portBindings := nat.PortMap{
		containerPort: []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", cfg.HostPort)},
		},
	}
	exposedPorts := nat.PortSet{containerPort: struct{}{}}

	// Volume mount
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

// EnsureNetwork creates the named network if it does not already exist.
func (dc *DockerClient) EnsureNetwork(ctx context.Context, name string) error {
	f := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: name})
	networks, err := dc.cli.NetworkList(ctx, dockernetwork.ListOptions{Filters: f})
	if err != nil {
		return err
	}
	for _, n := range networks {
		if n.Name == name {
			return nil
		}
	}
	_, err = dc.cli.NetworkCreate(ctx, name, dockernetwork.CreateOptions{Driver: "bridge"})
	return err
}

// VolumeCreate creates a named volume with labels.
func (dc *DockerClient) VolumeCreate(ctx context.Context, name string, labels map[string]string) error {
	_, err := dc.cli.VolumeCreate(ctx, volume.CreateOptions{Name: name, Labels: labels})
	return err
}

// VolumeRemove removes a volume; ignores "not found" errors.
func (dc *DockerClient) VolumeRemove(ctx context.Context, name string) error {
	err := dc.cli.VolumeRemove(ctx, name, true)
	if err != nil && !isNotFoundError(err) {
		return err
	}
	return nil
}

// NetworkDisconnect disconnects a container from a network; ignores errors.
func (dc *DockerClient) NetworkDisconnect(ctx context.Context, networkName, containerID string) {
	_ = dc.cli.NetworkDisconnect(ctx, networkName, containerID, true)
}

// ImageRef is a locally-available Docker image name and tag.
type ImageRef struct {
	Repo string
	Tag  string
}

// String returns the canonical "repo:tag" form.
func (r ImageRef) String() string { return r.Repo + ":" + r.Tag }

// ListImages returns all locally-available images whose repo matches any of the given repos.
// Returns nil, nil when no images are found or on any non-fatal error so callers degrade gracefully.
func (dc *DockerClient) ListImages(ctx context.Context, repos []string) ([]ImageRef, error) {
	if len(repos) == 0 {
		return nil, nil
	}

	seen := map[string]bool{}
	var result []ImageRef

	for _, repo := range repos {
		f := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: repo})
		images, err := dc.cli.ImageList(ctx, image.ListOptions{Filters: f})
		if err != nil {
			return nil, nil
		}
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if tag == "<none>:<none>" || tag == "" {
					continue
				}
				parts := strings.SplitN(tag, ":", 2)
				if len(parts) != 2 || parts[1] == "" {
					continue
				}
				ref := ImageRef{Repo: parts[0], Tag: parts[1]}
				key := ref.String()
				if !seen[key] {
					seen[key] = true
					result = append(result, ref)
				}
			}
		}
	}
	return result, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "No such") || strings.Contains(msg, "not found")
}

func isNotRunningError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "is not running") || strings.Contains(msg, "already stopped")
}
