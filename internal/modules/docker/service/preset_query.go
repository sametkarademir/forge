package service

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
)

// RemovePreset stops and removes the container and volume for a preset.
// mode controls whether the preset YAML is also deleted.
// Idempotent — missing resources are silently skipped.
// Legacy fallback: if no preset-labeled container exists, falls back to looking
// up a container by forge.project=<name> for pre-refactor containers.
func RemovePreset(ctx context.Context, name string, mode RemoveMode) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}

	presetExists := preset.Exists(name)

	// Try preset-labeled container first.
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		// No preset container — try legacy fallback.
		if legacy, legacyErr := dc.InspectByProject(ctx, name); legacyErr == nil &&
			legacy.Config.Labels["forge.managed"] == "true" {
			engineName := legacy.Config.Labels["forge.engine"]
			_ = dc.StopContainer(ctx, legacy.ID)
			if rmErr := dc.RemoveContainer(ctx, legacy.ID); rmErr != nil {
				return fmt.Errorf("remove legacy container: %w", rmErr)
			}
			_ = dc.VolumeRemove(ctx, VolumeName(name, engineName))
		}
		if !presetExists {
			return nil
		}
		if mode == RemoveAll {
			return preset.Delete(name)
		}
		return nil
	}

	// Found preset-labeled container.
	_ = dc.StopContainer(ctx, c.ID)
	if err := dc.RemoveContainer(ctx, c.ID); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	if err := dc.VolumeRemove(ctx, PresetVolumeName(name)); err != nil {
		return fmt.Errorf("remove volume: %w", err)
	}
	if mode == RemoveAll {
		if err := preset.Delete(name); err != nil {
			return fmt.Errorf("delete preset: %w", err)
		}
	}
	return nil
}

// ShowPreset returns the preset config (password masked) plus container state.
func ShowPreset(ctx context.Context, name string) (*PresetView, error) {
	p, err := preset.Load(name)
	if err != nil {
		return nil, err
	}

	view := &PresetView{Preset: p}

	dc, err := dockerclient.NewClient()
	if err != nil {
		view.Status = "unknown"
		return view, nil
	}

	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		view.Status = "not created"
		return view, nil
	}

	port, _ := strconv.Atoi(c.Config.Labels["forge.host_port"])
	view.HostPort = port

	if c.State.Running {
		view.Status = "running"
		if eng, ok := engines.Get(p.Engine); ok {
			raw := eng.ConnectionString("localhost", port, p.Username, p.Password, p.Database)
			view.DSN = maskPasswordInDSN(raw)
		}
	} else {
		view.Status = c.State.Status // "exited", "paused", etc.
	}

	return view, nil
}

// ListAll joins disk presets with Docker container state and returns one Row
// per managed resource (preset or container).
func ListAll(ctx context.Context) ([]Row, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}

	containers, err := dc.ListManaged(ctx)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	entries, err := preset.ListEntries()
	if err != nil {
		return nil, fmt.Errorf("list presets: %w", err)
	}

	// Partition containers into preset-labeled and legacy.
	containerByPreset := map[string]dockertypes.Container{}
	var unmapped []dockertypes.Container

	for _, c := range containers {
		if pname := c.Labels["forge.preset"]; pname != "" {
			containerByPreset[pname] = c
		} else {
			unmapped = append(unmapped, c)
		}
	}

	var rows []Row

	for _, e := range entries {
		c, hasContainer := containerByPreset[e.Name]
		delete(containerByPreset, e.Name) // mark matched even for invalid entries

		if e.Err != nil {
			rows = append(rows, Row{Name: e.Name, Status: "invalid"})
			continue
		}

		p := e.Preset
		row := Row{
			Name:      e.Name,
			Engine:    p.Engine,
			Image:     p.Image,
			CreatedAt: p.CreatedAt,
		}
		if !hasContainer {
			row.Status = "not created"
		} else {
			row.HostPort, _ = strconv.Atoi(c.Labels["forge.host_port"])
			if c.State == "running" {
				row.Status = "running"
			} else {
				row.Status = "stopped"
			}
		}
		rows = append(rows, row)
	}

	// Remaining preset-labeled containers with no matching preset file → orphaned.
	for pname, c := range containerByPreset {
		port, _ := strconv.Atoi(c.Labels["forge.host_port"])
		createdAt, _ := time.Parse(time.RFC3339, c.Labels["forge.created_at"])
		rows = append(rows, Row{
			Name:      pname,
			Engine:    c.Labels["forge.engine"],
			Image:     c.Image,
			HostPort:  port,
			Status:    "orphaned",
			CreatedAt: createdAt,
		})
	}

	// forge.managed=true containers without forge.preset → legacy.
	for _, c := range unmapped {
		engineName := c.Labels["forge.engine"]
		name := extractLegacyName(c, engineName)
		port, _ := strconv.Atoi(c.Labels["forge.host_port"])
		createdAt, _ := time.Parse(time.RFC3339, c.Labels["forge.created_at"])
		rows = append(rows, Row{
			Name:      name,
			Engine:    engineName,
			Image:     c.Image,
			HostPort:  port,
			Status:    "legacy",
			CreatedAt: createdAt,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

// LogsPreset returns a multiplexed stdout+stderr stream for the preset container.
// Callers should use stdcopy.StdCopy to demultiplex.
func LogsPreset(ctx context.Context, name string, follow bool) (io.ReadCloser, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("no container found for preset %q", name)
	}
	return dc.ContainerLogs(ctx, c.ID, follow)
}

// ConnString returns the unmasked DSN for a running preset container.
// Returns an error if the container is not running.
func ConnString(ctx context.Context, name string) (string, error) {
	p, err := preset.Load(name)
	if err != nil {
		return "", err
	}

	dc, err := dockerclient.NewClient()
	if err != nil {
		return "", err
	}

	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return "", fmt.Errorf("preset %q has no container — start with: forge docker run %s", name, name)
	}
	if !c.State.Running {
		return "", fmt.Errorf("preset %q is not running — start with: forge docker run %s", name, name)
	}

	port, _ := strconv.Atoi(c.Config.Labels["forge.host_port"])
	eng, ok := engines.Get(p.Engine)
	if !ok {
		return "", engines.ErrUnknownEngine(p.Engine)
	}
	return eng.ConnectionString("localhost", port, p.Username, p.Password, p.Database), nil
}

// extractLegacyName derives a human-readable name from a legacy container.
// Legacy container names follow the pattern "forge-<project>-<engine>".
func extractLegacyName(c dockertypes.Container, engineName string) string {
	if len(c.Names) == 0 {
		return c.Labels["forge.project"]
	}
	raw := strings.TrimPrefix(c.Names[0], "/forge-")
	if engineName != "" {
		raw = strings.TrimSuffix(raw, "-"+engineName)
	}
	return raw
}
