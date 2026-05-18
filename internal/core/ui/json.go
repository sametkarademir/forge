package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// EmitJSON marshals v as indented JSON to stdout.
func EmitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}
