package preset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// isSafe returns an error if path is a symlink that points outside dir.
// If path does not exist, nil is returned — there is nothing to reject.
func isSafe(dir, path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}

	// Dangling symlinks are rejected.
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("preset %q is a dangling symlink", filepath.Base(path))
	}

	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("resolve presets dir: %w", err)
	}

	if !strings.HasPrefix(real, realDir+string(filepath.Separator)) {
		return fmt.Errorf("preset %q is a symlink pointing outside the presets directory", filepath.Base(path))
	}
	return nil
}
