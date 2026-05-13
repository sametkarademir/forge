package preset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// overrideDir may be set in tests to redirect all file operations away from ~/.forge/presets.
var overrideDir string

// PresetsDir returns the directory where preset YAML files are stored.
func PresetsDir() (string, error) {
	if overrideDir != "" {
		return overrideDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".forge", "presets"), nil
}

// Path returns the full file path for the named preset.
func Path(name string) (string, error) {
	dir, err := PresetsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".yaml"), nil
}

// Save atomically writes p to ~/.forge/presets/<name>.yaml.
// It writes to a temp file, fsyncs, then renames over the destination.
func Save(p *Preset) error {
	dir, err := PresetsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create presets dir: %w", err)
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	tmp := filepath.Join(dir, "."+p.Name+".tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temp file: %w", err)
	}

	dest := filepath.Join(dir, p.Name+".yaml")
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("atomic rename: %w", err)
	}
	return nil
}

// Load reads and parses the preset with the given name.
// Returns an error wrapping "not found" if the file does not exist.
func Load(name string) (*Preset, error) {
	dir, err := PresetsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+".yaml")
	if err := isSafe(dir, path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("preset %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("read preset: %w", err)
	}
	var p Preset
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse preset %q: %w", name, err)
	}
	return &p, nil
}

// Exists reports whether a valid preset file exists for name.
// Returns false for symlinks pointing outside the presets dir.
func Exists(name string) bool {
	dir, err := PresetsDir()
	if err != nil {
		return false
	}
	path := filepath.Join(dir, name+".yaml")
	if err := isSafe(dir, path); err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Delete removes the preset YAML file for name. It is idempotent.
func Delete(name string) error {
	dir, err := PresetsDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name+".yaml")
	if err := isSafe(dir, path); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove preset %q: %w", name, err)
	}
	return nil
}

// Entry pairs a preset name with its parsed data or a parse error.
// The service layer uses this to surface "invalid" status for corrupt files.
type Entry struct {
	Name   string
	Preset *Preset
	Err    error
}

// ListEntries returns one Entry per YAML file in the presets directory.
// Dot-prefixed files and non-.yaml files are skipped. Symlinks pointing
// outside the directory are skipped silently. The directory not existing
// is treated as empty (returns nil, nil).
func ListEntries() ([]Entry, error) {
	dir, err := PresetsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read presets dir: %w", err)
	}

	var result []Entry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		presetName := strings.TrimSuffix(name, ".yaml")
		path := filepath.Join(dir, name)
		if err := isSafe(dir, path); err != nil {
			continue // silently skip symlink violations
		}
		p, parseErr := Load(presetName)
		result = append(result, Entry{Name: presetName, Preset: p, Err: parseErr})
	}
	return result, nil
}
