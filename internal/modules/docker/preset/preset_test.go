package preset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func useTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	overrideDir = dir
	t.Cleanup(func() { overrideDir = "" })
	return dir
}

// --- ValidateName ---

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// valid
		{"a", false},
		{"pg1", false},
		{"my-preset", false},
		{"0abc", false},
		{"a" + strings.Repeat("b", 31), false}, // 32 chars — max

		// invalid pattern
		{"", true},                            // empty
		{"-start", true},                      // leading hyphen
		{"UPPER", true},                       // uppercase
		{"under_score", true},                 // underscore
		{"has space", true},                   // space
		{"a!b", true},                         // special char
		{"a" + strings.Repeat("b", 32), true}, // 33 chars — exceeds max

		// reserved
		{"net", true},
		{"default", true},
		{"all", true},
		{"new", true},
		{"list", true},
		{"show", true},
		{"run", true},
		{"stop", true},
		{"reset", true},
		{"remove", true},
		{"logs", true},
		{"conn", true},
	}
	for _, tc := range tests {
		err := ValidateName(tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateName(%q): got err=%v, wantErr=%v", tc.name, err, tc.wantErr)
		}
	}
}

// --- Save / Load roundtrip ---

func samplePreset(name string) *Preset {
	return &Preset{
		SchemaVersion: 1,
		Name:          name,
		Engine:        "postgres",
		Image:         "postgres:16-alpine",
		Database:      "mydb",
		Username:      "forge",
		Password:      "s3cr3t",
		InternalPort:  5432,
		CreatedAt:     time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC),
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	useTestDir(t)
	p := samplePreset("wp-pg")
	if err := Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load("wp-pg")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != p.Name || got.Engine != p.Engine || got.Password != p.Password {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, p)
	}
}

func TestSaveIdempotent(t *testing.T) {
	useTestDir(t)
	p := samplePreset("idm")
	if err := Save(p); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	p.Image = "postgres:17-alpine"
	if err := Save(p); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := Load("idm")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Image != "postgres:17-alpine" {
		t.Errorf("second save did not overwrite: got image %q", got.Image)
	}
}

func TestSaveNoTempFileAfterSuccess(t *testing.T) {
	dir := useTestDir(t)
	p := samplePreset("clean")
	if err := Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	tmp := filepath.Join(dir, ".clean.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temp file %s still exists after successful save", tmp)
	}
}

func TestSaveLeftoverTempDoesNotBlockSave(t *testing.T) {
	dir := useTestDir(t)
	// Simulate a crash-leftover temp file
	tmp := filepath.Join(dir, ".pg.tmp")
	if err := os.WriteFile(tmp, []byte("garbage"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	p := samplePreset("pg")
	// Save must succeed even with a leftover .tmp
	if err := Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("leftover temp still exists after save: %v", err)
	}
}

// --- Load: not found & corrupt ---

func TestLoadNotFound(t *testing.T) {
	useTestDir(t)
	_, err := Load("ghost")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Load missing preset: want 'not found' error, got %v", err)
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := useTestDir(t)
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":\n\tnot: valid: yaml\n\t\tbad"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := Load("bad")
	if err == nil {
		t.Error("Load corrupt YAML: expected error, got nil")
	}
}

// --- Exists ---

func TestExists(t *testing.T) {
	useTestDir(t)
	if Exists("nope") {
		t.Error("Exists: expected false for non-existent preset")
	}
	if err := Save(samplePreset("pg")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists("pg") {
		t.Error("Exists: expected true after Save")
	}
}

// --- Delete ---

func TestDeleteIdempotent(t *testing.T) {
	useTestDir(t)
	if err := Delete("ghost"); err != nil {
		t.Errorf("Delete non-existent: got error %v, want nil", err)
	}
}

func TestDeleteRemovesFile(t *testing.T) {
	useTestDir(t)
	if err := Save(samplePreset("tobedel")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := Delete("tobedel"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if Exists("tobedel") {
		t.Error("Delete: preset still exists after Delete")
	}
}

// --- ListEntries ---

func TestListEntriesEmpty(t *testing.T) {
	useTestDir(t)
	entries, err := ListEntries()
	if err != nil {
		t.Fatalf("ListEntries on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListEntriesDirNotExist(t *testing.T) {
	// A non-existent overrideDir must return nil, nil — not an error.
	overrideDir = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { overrideDir = "" })

	entries, err := ListEntries()
	if err != nil {
		t.Fatalf("ListEntries for missing dir: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestListEntriesFiltersNonYAML(t *testing.T) {
	dir := useTestDir(t)
	// dot-prefixed and non-yaml files must be skipped
	_ = os.WriteFile(filepath.Join(dir, ".tmp"), []byte("x"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o600)
	if err := Save(samplePreset("real")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "real" {
		t.Errorf("expected 1 entry 'real', got %v", entries)
	}
}

func TestListEntriesIncludesInvalid(t *testing.T) {
	dir := useTestDir(t)
	_ = os.WriteFile(filepath.Join(dir, "corrupt.yaml"), []byte(":\n\tbad:\n\t\t: yaml"), 0o600)
	if err := Save(samplePreset("good")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (good + corrupt), got %d", len(entries))
	}
	good, bad := 0, 0
	for _, e := range entries {
		if e.Err == nil {
			good++
		} else {
			bad++
		}
	}
	if good != 1 || bad != 1 {
		t.Errorf("expected 1 valid + 1 invalid entry, got good=%d bad=%d", good, bad)
	}
}

// --- Symlink rejection ---

func TestSymlinkRejected(t *testing.T) {
	dir := useTestDir(t)

	// Create a real file outside the presets dir.
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("setup outside file: %v", err)
	}

	// Symlink it into the presets dir.
	link := filepath.Join(dir, "evil.yaml")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Load must reject it.
	_, err := Load("evil")
	if err == nil {
		t.Fatal("Load symlink: expected error, got nil")
	}

	// Exists must return false.
	if Exists("evil") {
		t.Error("Exists symlink: expected false")
	}

	// ListEntries must skip it silently.
	entries, err := ListEntries()
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	for _, e := range entries {
		if e.Name == "evil" {
			t.Error("ListEntries included symlink entry")
		}
	}
}
