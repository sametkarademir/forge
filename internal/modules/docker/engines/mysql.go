package engines

import (
	"fmt"
	"strings"
)

type mysql struct{}

func init() { Register(&mysql{}) }

func (m *mysql) Name() string            { return "mysql" }
func (m *mysql) DefaultImage() string    { return "mysql:8.4" }
func (m *mysql) ImageRepos() []string    { return []string{"mysql"} }
func (m *mysql) DefaultPort() int        { return 3306 }
func (m *mysql) DataDir(_ string) string { return "/var/lib/mysql" }
func (m *mysql) PasswordEnvKey() string  { return "MYSQL_PASSWORD" }

func (m *mysql) EnvVars(user, password, db string) map[string]string {
	return map[string]string{
		"MYSQL_USER":          user,
		"MYSQL_PASSWORD":      password,
		"MYSQL_DATABASE":      db,
		"MYSQL_ROOT_PASSWORD": password,
	}
}

func (m *mysql) ConnectionInfo(a ConnArgs) ConnInfo {
	raw := fmt.Sprintf("mysql://%s:%s@%s:%d/%s", a.User, a.Password, a.Host, a.HostPort, a.Database)
	masked := strings.Replace(raw, ":"+a.Password+"@", ":****@", 1)
	return ConnInfo{Primary: raw, MaskedPrimary: masked}
}

func (m *mysql) Cmd(_ string) []string                  { return nil }
func (m *mysql) ValidatePassword(password string) error { return nil }
