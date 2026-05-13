package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
)

// pullOpts returns a zero-value PullOptions for convenience.
func pullOpts() dockerimage.PullOptions { return dockerimage.PullOptions{} }

// PullImage explicitly pulls ref, surfacing errors (unlike RunContainer's silent pull).
func (dc *DockerClient) PullImage(ctx context.Context, ref string) error {
	reader, err := dc.cli.ImagePull(ctx, ref, pullOpts())
	if err != nil {
		return fmt.Errorf("pull image %s: %w", ref, err)
	}
	_, _ = io.Copy(io.Discard, reader)
	return reader.Close()
}

// ImageExists reports whether ref is present in the local image store.
func (dc *DockerClient) ImageExists(ctx context.Context, ref string) (bool, error) {
	f := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: ref})
	images, err := dc.cli.ImageList(ctx, dockerimage.ListOptions{Filters: f})
	if err != nil {
		return false, fmt.Errorf("list images: %w", err)
	}
	return len(images) > 0, nil
}

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
		imgs, err := dc.cli.ImageList(ctx, dockerimage.ListOptions{Filters: f})
		if err != nil {
			return nil, nil
		}
		for _, img := range imgs {
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
