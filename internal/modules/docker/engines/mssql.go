package engines

import (
	"fmt"
	"strings"
	"unicode"
)

type mssql struct{}

func init() { Register(&mssql{}) }

func (m *mssql) Name() string            { return "mssql" }
func (m *mssql) DefaultImage() string    { return "mcr.microsoft.com/mssql/server:2022-latest" }
func (m *mssql) ImageRepos() []string    { return []string{"mcr.microsoft.com/mssql/server"} }
func (m *mssql) DefaultPort() int        { return 1433 }
func (m *mssql) DataDir(_ string) string { return "/var/opt/mssql" }
func (m *mssql) PasswordEnvKey() string  { return "SA_PASSWORD" }

func (m *mssql) EnvVars(user, password, db string) map[string]string {
	return map[string]string{
		"ACCEPT_EULA": "Y",
		"SA_PASSWORD": password,
		"MSSQL_PID":   "Developer",
	}
}

func (m *mssql) ConnectionString(host string, hostPort int, user, password, db string) string {
	return fmt.Sprintf(
		"Server=%s,%d;Database=%s;User Id=sa;Password=%s;TrustServerCertificate=true",
		host, hostPort, db, password,
	)
}

// ValidatePassword enforces SQL Server 2019+ SA_PASSWORD complexity rules.
func (m *mssql) ValidatePassword(password string) error {
	var failures []string

	if len(password) < 8 {
		failures = append(failures, "minimum 8 characters")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	special := "!@#$%^&*()-_+=[]{}|;:,.<>?"
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case strings.ContainsRune(special, r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		failures = append(failures, "missing uppercase letter")
	}
	if !hasLower {
		failures = append(failures, "missing lowercase letter")
	}
	if !hasDigit {
		failures = append(failures, "missing digit")
	}
	if !hasSpecial {
		failures = append(failures, "missing special character (one of: !@#$%^&*()-_+=[]{}|;:,.<>?)")
	}

	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, ", "))
	}
	return nil
}
