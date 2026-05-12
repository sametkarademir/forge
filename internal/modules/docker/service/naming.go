package service

// ContainerName returns the deterministic container name for a project+engine pair.
func ContainerName(project, engine string) string {
	return "forge-" + project + "-" + engine
}

// VolumeName returns the deterministic volume name for a project+engine pair.
func VolumeName(project, engine string) string {
	return "forge-" + project + "-" + engine + "-data"
}

// NetworkName returns the shared forge Docker network name.
func NetworkName() string {
	return "forge-net"
}
