package client

import (
	"context"
	"fmt"
	"strings"

	dockerclient "github.com/docker/docker/client"
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

// ImageRef is a locally-available Docker image name and tag.
type ImageRef struct {
	Repo string
	Tag  string
}

// String returns the canonical "repo:tag" form.
func (r ImageRef) String() string { return r.Repo + ":" + r.Tag }

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
