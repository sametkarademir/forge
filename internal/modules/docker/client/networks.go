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

// RemoveNetwork removes a named network; ignores "not found" errors.
func (dc *DockerClient) RemoveNetwork(ctx context.Context, name string) error {
	f := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: name})
	networks, err := dc.cli.NetworkList(ctx, dockernetwork.ListOptions{Filters: f})
	if err != nil {
		return err
	}
	for _, n := range networks {
		if n.Name == name {
			return dc.cli.NetworkRemove(ctx, n.ID)
		}
	}
	return nil // not found — idempotent success
}

// IsNetworkEmpty reports whether the named network has no containers connected.
func (dc *DockerClient) IsNetworkEmpty(ctx context.Context, name string) (bool, error) {
	f := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: name})
	networks, err := dc.cli.NetworkList(ctx, dockernetwork.ListOptions{Filters: f})
	if err != nil {
		return false, err
	}
	for _, n := range networks {
		if n.Name == name {
			return len(n.Containers) == 0, nil
		}
	}
	return true, nil // network doesn't exist
}
