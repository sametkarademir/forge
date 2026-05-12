package engines

import "fmt"

type postgres struct{}

func init() { Register(&postgres{}) }

func (p *postgres) Name() string           { return "postgres" }
func (p *postgres) DefaultImage() string   { return "postgres:16-alpine" }
func (p *postgres) DefaultPort() int       { return 5432 }
func (p *postgres) DataDir() string        { return "/var/lib/postgresql/data" }
func (p *postgres) PasswordEnvKey() string { return "POSTGRES_PASSWORD" }

func (p *postgres) EnvVars(user, password, db string) map[string]string {
	return map[string]string{
		"POSTGRES_USER":     user,
		"POSTGRES_PASSWORD": password,
		"POSTGRES_DB":       db,
	}
}

func (p *postgres) ConnectionString(host string, hostPort int, user, password, db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, hostPort, db)
}

func (p *postgres) ValidatePassword(password string) error { return nil }
