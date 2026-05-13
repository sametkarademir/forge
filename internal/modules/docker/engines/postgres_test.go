package engines

import "testing"

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
