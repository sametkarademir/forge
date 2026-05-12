package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldCheckNoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if !ShouldCheck() {
		t.Error("ShouldCheck() = false, want true when no throttle file exists")
	}
}

func TestShouldCheckFreshFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	RecordCheck()

	if ShouldCheck() {
		t.Error("ShouldCheck() = true, want false immediately after RecordCheck")
	}
}

func TestShouldCheckExpiredFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path := filepath.Join(dir, ".forge", "update_check")
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	_ = f.Close()

	// Backdate the file's mtime by 25 hours.
	old := time.Now().Add(-25 * time.Hour)
	_ = os.Chtimes(path, old, old)

	if !ShouldCheck() {
		t.Error("ShouldCheck() = false, want true when throttle file is older than 24h")
	}
}
