package engines

import "fmt"

type mysql struct{}

func init() { Register(&mysql{}) }

func (m *mysql) Name() string           { return "mysql" }
func (m *mysql) DefaultImage() string   { return "mysql:8.4" }
func (m *mysql) DefaultPort() int       { return 3306 }
func (m *mysql) DataDir() string        { return "/var/lib/mysql" }
func (m *mysql) PasswordEnvKey() string { return "MYSQL_PASSWORD" }

func (m *mysql) EnvVars(user, password, db string) map[string]string {
	return map[string]string{
		"MYSQL_USER":          user,
		"MYSQL_PASSWORD":      password,
		"MYSQL_DATABASE":      db,
		"MYSQL_ROOT_PASSWORD": password,
	}
}

func (m *mysql) ConnectionString(host string, hostPort int, user, password, db string) string {
	return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", user, password, host, hostPort, db)
}

func (m *mysql) ValidatePassword(password string) error { return nil }
