package ui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// CopyToClipboard writes text to the system clipboard.
// Only supported on macOS (uses pbcopy).
func CopyToClipboard(text string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("clipboard copy is only supported on macOS")
	}
	cmd := exec.Command("pbcopy")
	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}
	if _, err := fmt.Fprint(in, text); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	_ = in.Close()
	return cmd.Wait()
}
