package client

import (
	"context"

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
