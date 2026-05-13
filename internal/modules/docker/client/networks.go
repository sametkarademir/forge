package client

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	dockernetwork "github.com/docker/docker/api/types/network"
)

// EnsureNetwork creates the named bridge network if it does not already exist.
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

// NetworkDisconnect disconnects a container from a network; errors are swallowed.
func (dc *DockerClient) NetworkDisconnect(ctx context.Context, networkName, containerID string) {
	_ = dc.cli.NetworkDisconnect(ctx, networkName, containerID, true)
}
