package service

// ContainerName returns the deterministic container name for a project+engine pair.
func ContainerName(project, engine string) string {
	return "forge-" + project + "-" + engine
}

// VolumeName returns the deterministic volume name for a project+engine pair.
func VolumeName(project, engine string) string {
	return "forge-" + project + "-" + engine + "-data"
}

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
