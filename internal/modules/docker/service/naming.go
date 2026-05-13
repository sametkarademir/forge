package service

// PresetContainerName returns the deterministic container name for a preset.
func PresetContainerName(presetName string) string {
	return "forge-" + presetName
}

// PresetVolumeName returns the deterministic volume name for a preset.
func PresetVolumeName(presetName string) string {
	return "forge-" + presetName + "-data"
}

// NetworkName returns the shared forge Docker network name.
func NetworkName() string {
	return "forge-net"
}
