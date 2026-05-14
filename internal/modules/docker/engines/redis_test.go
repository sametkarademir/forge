package engines

import (
	"strings"
	"testing"
)

func TestRedisConnectionInfo(t *testing.T) {
	r := &redis{}
	ci := r.ConnectionInfo(ConnArgs{Host: "localhost", HostPort: 6379, Password: "s3cr3t", Database: "0"})

	want := "redis://:s3cr3t@localhost:6379/0"
	if ci.Primary != want {
		t.Errorf("Primary = %q, want %q", ci.Primary, want)
	}
	if strings.Contains(ci.MaskedPrimary, "s3cr3t") {
		t.Errorf("MaskedPrimary should not contain password, got %q", ci.MaskedPrimary)
	}
	if !strings.Contains(ci.MaskedPrimary, "****") {
		t.Errorf("MaskedPrimary should contain ****, got %q", ci.MaskedPrimary)
	}
	if len(ci.Endpoints) == 0 {
		t.Fatal("Endpoints should be non-nil for redis")
	}
	if ci.Endpoints[0].Label != "CLI" {
		t.Errorf("Endpoints[0].Label = %q, want %q", ci.Endpoints[0].Label, "CLI")
	}
	if strings.Contains(ci.Endpoints[0].Value, "s3cr3t") {
		t.Errorf("CLI endpoint must not contain password, got %q", ci.Endpoints[0].Value)
	}
}

func TestRedisConnectionInfoInvalidDBFallsBackToZero(t *testing.T) {
	r := &redis{}
	cases := []string{"not-a-number", "16", "-1", ""}
	for _, db := range cases {
		ci := r.ConnectionInfo(ConnArgs{Host: "h", HostPort: 6379, Password: "p", Database: db})
		if !strings.HasSuffix(ci.Primary, "/0") {
			t.Errorf("Database=%q: Primary = %q, want suffix /0", db, ci.Primary)
		}
	}
}

func TestRedisConnectionInfoValidDBIndex(t *testing.T) {
	r := &redis{}
	ci := r.ConnectionInfo(ConnArgs{Host: "localhost", HostPort: 6379, Password: "p", Database: "5"})
	if !strings.HasSuffix(ci.Primary, "/5") {
		t.Errorf("Primary = %q, want suffix /5", ci.Primary)
	}
}

func TestRedisCmdEmbedsPassword(t *testing.T) {
	r := &redis{}
	cmd := r.Cmd("s3cr3t")

	if len(cmd) == 0 || cmd[0] != "redis-server" {
		t.Errorf("cmd[0] = %q, want %q", cmd[0], "redis-server")
	}
	assertContainsSequence(t, cmd, "--requirepass", "s3cr3t")
	assertContainsSequence(t, cmd, "--save", "20 1")
	assertContainsSequence(t, cmd, "--appendonly", "yes")
	assertContainsSequence(t, cmd, "--loglevel", "warning")
}

func TestRedisValidatePassword(t *testing.T) {
	r := &redis{}
	if err := r.ValidatePassword(""); err == nil {
		t.Error("ValidatePassword(\"\") should return error")
	}
	if err := r.ValidatePassword("anyvalue"); err != nil {
		t.Errorf("ValidatePassword(\"anyvalue\") = %v, want nil", err)
	}
}

// assertContainsSequence checks that flag appears in args followed immediately by value.
func assertContainsSequence(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i, a := range args {
		if a == flag {
			if i+1 >= len(args) {
				t.Errorf("%q found at end of slice, no value follows", flag)
				return
			}
			if args[i+1] != value {
				t.Errorf("after %q: got %q, want %q", flag, args[i+1], value)
			}
			return
		}
	}
	t.Errorf("%q not found in %v", flag, args)
}
