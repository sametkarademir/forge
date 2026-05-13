package preset

import (
	"fmt"
	"regexp"
)

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

var reservedNames = map[string]bool{
	"net": true, "default": true, "all": true, "new": true,
	"list": true, "show": true, "run": true, "stop": true,
	"reset": true, "remove": true, "logs": true, "conn": true,
}

// ValidateName returns an error if name does not match the preset naming rules
// or is a reserved CLI subcommand name.
func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("preset name %q must match ^[a-z0-9][a-z0-9-]{0,31}$ (lowercase, 1–32 chars)", name)
	}
	if reservedNames[name] {
		return fmt.Errorf("preset name %q is reserved", name)
	}
	return nil
}
