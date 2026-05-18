package engines

import (
	"fmt"
	"strconv"
	"strings"
)

type postgres struct{}

func init() { Register(&postgres{}) }

func (p *postgres) Name() string           { return "postgres" }
func (p *postgres) DefaultImage() string   { return "postgres:16-alpine" }
func (p *postgres) ImageRepos() []string   { return []string{"postgres"} }
func (p *postgres) DefaultPort() int       { return 5432 }
func (p *postgres) PasswordEnvKey() string { return "POSTGRES_PASSWORD" }

// DataDir returns the volume mount target inside the container.
// PG 18+ changed the on-disk layout: data lives under /var/lib/postgresql/<major>/docker/,
// so we mount the parent directory. PG 17 and below use the old /var/lib/postgresql/data path
// directly, which avoids Docker creating a shadowing anonymous volume from the image's VOLUME
// declaration.
func (p *postgres) DataDir(image string) string {
	if pgMajorVersion(image) >= 18 {
		return "/var/lib/postgresql"
	}
	return "/var/lib/postgresql/data"
}

func (p *postgres) EnvVars(user, password, db string) map[string]string {
	return map[string]string{
		"POSTGRES_USER":     user,
		"POSTGRES_PASSWORD": password,
		"POSTGRES_DB":       db,
	}
}

func (p *postgres) ConnectionInfo(a ConnArgs) ConnInfo {
	raw := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", a.User, a.Password, a.Host, a.HostPort, a.Database)
	masked := strings.Replace(raw, ":"+a.Password+"@", ":****@", 1)
	return ConnInfo{Primary: raw, MaskedPrimary: masked}
}

func (p *postgres) Cmd(_ string) []string                  { return nil }
func (p *postgres) ValidatePassword(password string) error { return nil }

func (p *postgres) ShellCmd(user, _, db string) []string {
	return []string{"psql", "-U", user, "-d", db}
}

// pgMajorVersion parses the major version number from an image tag such as
// "postgres:18-alpine", "postgres:17.2", or "postgis/postgis:18-3.5-alpine".
// Returns 0 if the version cannot be determined (e.g. "latest").
func pgMajorVersion(image string) int {
	tag := image
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		tag = image[idx+1:]
	}
	// Take the leading numeric segment before any '-' or '.'
	end := strings.IndexAny(tag, "-.")
	if end == -1 {
		end = len(tag)
	}
	v, err := strconv.Atoi(tag[:end])
	if err != nil {
		return 0
	}
	return v
}
