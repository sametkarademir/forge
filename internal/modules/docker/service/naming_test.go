package service

import "testing"

func TestContainerName(t *testing.T) {
	tests := []struct {
		project, engine, want string
	}{
		{"myapp", "postgres", "forge-myapp-postgres"},
		{"api", "mysql", "forge-api-mysql"},
		{"svc", "mssql", "forge-svc-mssql"},
	}
	for _, tc := range tests {
		if got := ContainerName(tc.project, tc.engine); got != tc.want {
			t.Errorf("ContainerName(%q, %q) = %q, want %q", tc.project, tc.engine, got, tc.want)
		}
	}
}

func TestVolumeName(t *testing.T) {
	tests := []struct {
		project, engine, want string
	}{
		{"myapp", "postgres", "forge-myapp-postgres-data"},
		{"api", "mysql", "forge-api-mysql-data"},
	}
	for _, tc := range tests {
		if got := VolumeName(tc.project, tc.engine); got != tc.want {
			t.Errorf("VolumeName(%q, %q) = %q, want %q", tc.project, tc.engine, got, tc.want)
		}
	}
}

func TestNetworkName(t *testing.T) {
	if got := NetworkName(); got != "forge-net" {
		t.Errorf("NetworkName() = %q, want %q", got, "forge-net")
	}
}
