package updater

import (
	"os"
	"path/filepath"
	"time"
)

const checkInterval = 24 * time.Hour

func ShouldCheck() bool {
	info, err := os.Stat(throttlePath())
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > checkInterval
}

func RecordCheck() {
	path := throttlePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	if err == nil {
		_ = f.Close()
	}
}

func throttlePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".forge", "update_check")
}
