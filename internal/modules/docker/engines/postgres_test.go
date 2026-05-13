package engines

import (
	"strings"
	"testing"
)

func TestPostgresConnectionInfo(t *testing.T) {
	p := &postgres{}
	ci := p.ConnectionInfo(ConnArgs{Host: "localhost", HostPort: 5432, User: "app", Password: "s3cr3t", Database: "mydb"})
	wantPrimary := "postgres://app:s3cr3t@localhost:5432/mydb"
	if ci.Primary != wantPrimary {
		t.Errorf("Primary = %q, want %q", ci.Primary, wantPrimary)
	}
	if strings.Contains(ci.MaskedPrimary, "s3cr3t") {
		t.Errorf("MaskedPrimary should not contain password, got %q", ci.MaskedPrimary)
	}
	if !strings.Contains(ci.MaskedPrimary, "****") {
		t.Errorf("MaskedPrimary should contain ****, got %q", ci.MaskedPrimary)
	}
	if ci.Endpoints != nil {
		t.Errorf("Endpoints should be nil for postgres, got %v", ci.Endpoints)
	}
}

func TestPgMajorVersion(t *testing.T) {
	cases := []struct {
		image string
		want  int
	}{
		{"postgres:18-alpine", 18},
		{"postgres:18.1", 18},
		{"postgres:17-alpine", 17},
		{"postgres:16", 16},
		{"postgis/postgis:18-3.5-alpine", 18},
		{"postgis/postgis:16-3.4", 16},
		{"postgres:latest", 0},
		{"postgres", 0},
	}
	for _, tc := range cases {
		got := pgMajorVersion(tc.image)
		if got != tc.want {
			t.Errorf("pgMajorVersion(%q) = %d, want %d", tc.image, got, tc.want)
		}
	}
}

func TestPostgresDataDir(t *testing.T) {
	p := &postgres{}
	cases := []struct {
		image string
		want  string
	}{
		{"postgres:18-alpine", "/var/lib/postgresql"},
		{"postgres:18.1", "/var/lib/postgresql"},
		{"postgis/postgis:18-3.5-alpine", "/var/lib/postgresql"},
		{"postgres:17-alpine", "/var/lib/postgresql/data"},
		{"postgres:16", "/var/lib/postgresql/data"},
		{"postgres:latest", "/var/lib/postgresql/data"},
	}
	for _, tc := range cases {
		got := p.DataDir(tc.image)
		if got != tc.want {
			t.Errorf("DataDir(%q) = %q, want %q", tc.image, got, tc.want)
		}
	}
}
