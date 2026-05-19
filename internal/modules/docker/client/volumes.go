package client

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

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

// ListManagedVolumes returns all volumes carrying the forge.managed=true label.
func (dc *DockerClient) ListManagedVolumes(ctx context.Context) ([]*volume.Volume, error) {
	f := filters.NewArgs(filters.KeyValuePair{Key: "label", Value: "forge.managed=true"})
	resp, err := dc.cli.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	return resp.Volumes, nil
}
