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
	"github.com/sametkarademir/forge/internal/core/logger"
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
			_ = dc.VolumeRemove(ctx, "forge-"+name+"-"+engineName+"-data")
		}
		if !presetExists {
			logger.Warn(fmt.Sprintf("no preset named %q — nothing to remove", name))
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
			ci := eng.ConnectionInfo(engines.ConnArgs{
				Host:       "localhost",
				HostPort:   port,
				User:       p.Username,
				Password:   p.Password,
				Database:   p.Database,
				Options:    p.Options,
				ExtraPorts: extraPortsFromLabels(eng, c.Config.Image, p.Options, c.Config.Labels),
			})
			view.Primary = ci.MaskedPrimary
			view.Endpoints = ci.Endpoints
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
func LogsPreset(ctx context.Context, name string, opts dockerclient.LogsOptions) (io.ReadCloser, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("no container found for preset %q", name)
	}
	return dc.ContainerLogs(ctx, c.ID, opts)
}

// GetPresetHostPort returns the host port assigned to a running preset container.
func GetPresetHostPort(ctx context.Context, name string) (int, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return 0, err
	}
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("preset %q has no container — start with: forge docker run %s", name, name)
	}
	port, _ := strconv.Atoi(c.Config.Labels["forge.host_port"])
	return port, nil
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
	ci := eng.ConnectionInfo(engines.ConnArgs{
		Host:       "localhost",
		HostPort:   port,
		User:       p.Username,
		Password:   p.Password,
		Database:   p.Database,
		Options:    p.Options,
		ExtraPorts: extraPortsFromLabels(eng, c.Config.Image, p.Options, c.Config.Labels),
	})
	return ci.Primary, nil
}

// ConnView holds both the unmasked and masked primary DSN plus any additional endpoints.
type ConnView struct {
	Primary       string
	MaskedPrimary string
	Endpoints     []engines.Endpoint
}

// ConnView returns full connection info for a running preset container.
// Returns an error if the container is not running.
func GetConnView(ctx context.Context, name string) (*ConnView, error) {
	p, err := preset.Load(name)
	if err != nil {
		return nil, err
	}
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}
	c, err := dc.InspectByPreset(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("preset %q has no container — start with: forge docker run %s", name, name)
	}
	if !c.State.Running {
		return nil, fmt.Errorf("preset %q is not running — start with: forge docker run %s", name, name)
	}
	port, _ := strconv.Atoi(c.Config.Labels["forge.host_port"])
	eng, ok := engines.Get(p.Engine)
	if !ok {
		return nil, engines.ErrUnknownEngine(p.Engine)
	}
	ci := eng.ConnectionInfo(engines.ConnArgs{
		Host:       "localhost",
		HostPort:   port,
		User:       p.Username,
		Password:   p.Password,
		Database:   p.Database,
		Options:    p.Options,
		ExtraPorts: extraPortsFromLabels(eng, c.Config.Image, p.Options, c.Config.Labels),
	})
	return &ConnView{
		Primary:       ci.Primary,
		MaskedPrimary: ci.MaskedPrimary,
		Endpoints:     ci.Endpoints,
	}, nil
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

// PruneTarget is one managed resource that prune would remove.
type PruneTarget struct {
	Kind   string // "container" or "volume"
	Name   string // human-readable name
	ID     string // Docker ID (containers only)
	Reason string // why it is a prune candidate
}

// FindOrphans returns managed containers and volumes that have no matching preset file.
// Orphaned: forge.preset label set but no preset YAML.
// Legacy: no forge.preset label (pre-refactor containers).
// Dangling volumes: forge.managed=true but preset YAML and container are both gone.
func FindOrphans(ctx context.Context) ([]PruneTarget, error) {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return nil, err
	}

	containers, err := dc.ListManaged(ctx)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	volumes, err := dc.ListManagedVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	var targets []PruneTarget
	activeVolumeNames := map[string]bool{}

	for _, c := range containers {
		cname := ""
		if len(c.Names) > 0 {
			cname = strings.TrimPrefix(c.Names[0], "/")
		}
		if pname := c.Labels["forge.preset"]; pname != "" {
			if !preset.Exists(pname) {
				targets = append(targets, PruneTarget{
					Kind:   "container",
					Name:   cname,
					ID:     c.ID,
					Reason: fmt.Sprintf("orphaned — no preset %q", pname),
				})
				continue
			}
		} else {
			targets = append(targets, PruneTarget{
				Kind:   "container",
				Name:   cname,
				ID:     c.ID,
				Reason: "legacy — pre-v2 container without forge.preset label",
			})
			continue
		}
		// Container has a valid preset — its volume is in use.
		if vname := c.Labels["forge.preset"]; vname != "" {
			activeVolumeNames["forge-"+vname+"-data"] = true
		}
	}

	// Volumes with no matching container.
	for _, v := range volumes {
		if !activeVolumeNames[v.Name] {
			pname := v.Labels["forge.preset"]
			if pname != "" && preset.Exists(pname) {
				continue // preset exists and container may just be stopped — not dangling
			}
			reason := "dangling — no matching container or preset"
			if pname != "" {
				reason = fmt.Sprintf("dangling — preset %q no longer exists", pname)
			}
			targets = append(targets, PruneTarget{
				Kind:   "volume",
				Name:   v.Name,
				Reason: reason,
			})
		}
	}

	return targets, nil
}

// Prune removes the given targets. Targets should come from FindOrphans.
func Prune(ctx context.Context, targets []PruneTarget) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}
	for _, t := range targets {
		switch t.Kind {
		case "container":
			_ = dc.StopContainer(ctx, t.ID)
			if err := dc.RemoveContainer(ctx, t.ID); err != nil {
				return fmt.Errorf("remove container %s: %w", t.Name, err)
			}
			logger.Success("Removed container " + t.Name)
		case "volume":
			if err := dc.VolumeRemove(ctx, t.Name); err != nil {
				return fmt.Errorf("remove volume %s: %w", t.Name, err)
			}
			logger.Success("Removed volume " + t.Name)
		}
	}
	return nil
}

// PruneNetwork removes forge-net if it has no connected containers.
func PruneNetwork(ctx context.Context, networkName string) error {
	dc, err := dockerclient.NewClient()
	if err != nil {
		return err
	}
	empty, err := dc.IsNetworkEmpty(ctx, networkName)
	if err != nil {
		return fmt.Errorf("check network: %w", err)
	}
	if !empty {
		return fmt.Errorf("network %q still has connected containers — stop all presets first", networkName)
	}
	if err := dc.RemoveNetwork(ctx, networkName); err != nil {
		return fmt.Errorf("remove network %s: %w", networkName, err)
	}
	logger.Success("Removed network " + networkName)
	return nil
}
