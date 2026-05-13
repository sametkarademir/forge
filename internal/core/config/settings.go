package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Source indicates whether a setting value came from the config file or the compiled default.
type Source string

const (
	SourceDefault Source = "default"
	SourceUser    Source = "user"
)

// Setting describes one configurable key and how to obtain its baseline default.
type Setting struct {
	Key          string
	DefaultValue func() string
}

// Row is a resolved setting ready for display.
type Row struct {
	Key    string
	Value  string
	Source Source
}

// Snapshot resolves the current value and source for each setting.
// Source is "user" when the key is explicitly present in the config file; "default" otherwise.
func Snapshot(settings []Setting) []Row {
	rows := make([]Row, 0, len(settings))
	for _, s := range settings {
		var row Row
		row.Key = s.Key
		if viper.InConfig(s.Key) {
			row.Value = viper.GetString(s.Key)
			row.Source = SourceUser
		} else {
			row.Value = s.DefaultValue()
			row.Source = SourceDefault
		}
		rows = append(rows, row)
	}
	return rows
}

// ValidateKey returns the Setting whose Key matches key (exact or with prefix prepended).
// prefix is the module prefix, e.g. "docker". Returns an error if the key is not in the list.
func ValidateKey(prefix string, settings []Setting, key string) (Setting, error) {
	normalised := key
	if !strings.HasPrefix(key, prefix+".") {
		normalised = prefix + "." + key
	}
	for _, s := range settings {
		if s.Key == normalised {
			return s, nil
		}
	}
	return Setting{}, fmt.Errorf(
		"unknown config key %q — run 'forge docker config show' to see available keys",
		key,
	)
}
