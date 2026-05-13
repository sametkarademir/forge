package service

import "testing"

func TestPresetContainerName(t *testing.T) {
	tests := []struct {
		preset, want string
	}{
		{"myapp", "forge-myapp"},
		{"wp-pg", "forge-wp-pg"},
	}
	for _, tc := range tests {
		if got := PresetContainerName(tc.preset); got != tc.want {
			t.Errorf("PresetContainerName(%q) = %q, want %q", tc.preset, got, tc.want)
		}
	}
}

func TestPresetVolumeName(t *testing.T) {
	tests := []struct {
		preset, want string
	}{
		{"myapp", "forge-myapp-data"},
		{"wp-pg", "forge-wp-pg-data"},
	}
	for _, tc := range tests {
		if got := PresetVolumeName(tc.preset); got != tc.want {
			t.Errorf("PresetVolumeName(%q) = %q, want %q", tc.preset, got, tc.want)
		}
	}
}

func TestNetworkName(t *testing.T) {
	if got := NetworkName(); got != "forge-net" {
		t.Errorf("NetworkName() = %q, want %q", got, "forge-net")
	}
}
