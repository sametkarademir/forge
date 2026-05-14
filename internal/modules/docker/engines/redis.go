package engines

import (
	"fmt"
	"strconv"
	"strings"
)

type redis struct{}

func init() { Register(&redis{}) }

func (r *redis) Name() string            { return "redis" }
func (r *redis) DefaultImage() string    { return "redis:7-alpine" }
func (r *redis) ImageRepos() []string    { return []string{"redis"} }
func (r *redis) DefaultPort() int        { return 6379 }
func (r *redis) DataDir(_ string) string { return "/data" }

// PasswordEnvKey returns empty: the official redis image does not accept auth via env var.
// Auth is configured through Cmd (--requirepass).
func (r *redis) PasswordEnvKey() string { return "" }

// EnvVars returns no environment variables. Redis auth is set via --requirepass in Cmd.
// The user/db arguments are accepted to satisfy the Engine interface and are unused.
func (r *redis) EnvVars(_, _, _ string) map[string]string {
	return map[string]string{}
}

// Cmd returns the redis-server invocation with safe single-node defaults:
//   - --requirepass enforces authentication
//   - --save 20 1 creates an RDB snapshot every 20 s if at least 1 key changed
//   - --appendonly yes adds AOF persistence alongside RDB for maximum durability
//   - --loglevel warning reduces log noise
func (r *redis) Cmd(password string) []string {
	return []string{
		"redis-server",
		"--requirepass", password,
		"--save", "20 1",
		"--appendonly", "yes",
		"--loglevel", "warning",
	}
}

// ConnectionInfo builds a redis:// URL. The pre-ACL Redis URL has no username:
// redis://:<password>@host:port/<dbindex>
// Database is interpreted as a numeric DB index (0–15); falls back to 0.
func (r *redis) ConnectionInfo(a ConnArgs) ConnInfo {
	dbIndex := 0
	if n, err := strconv.Atoi(a.Database); err == nil && n >= 0 && n <= 15 {
		dbIndex = n
	}
	raw := fmt.Sprintf("redis://:%s@%s:%d/%d", a.Password, a.Host, a.HostPort, dbIndex)
	masked := strings.Replace(raw, ":"+a.Password+"@", ":****@", 1)
	return ConnInfo{
		Primary:       raw,
		MaskedPrimary: masked,
		Endpoints: []Endpoint{
			// Password intentionally omitted per Endpoint contract (no secrets in values).
			// Retrieve the full connection string with `forge docker conn <preset>`.
			{Label: "CLI", Value: fmt.Sprintf("redis-cli -h %s -p %d -a <password>", a.Host, a.HostPort)},
		},
	}
}

func (r *redis) ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}
