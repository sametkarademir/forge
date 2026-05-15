package engines

import (
	"strings"
	"testing"
)

func TestRabbitMQConnectionInfo(t *testing.T) {
	r := &rabbitmq{}
	ci := r.ConnectionInfo(ConnArgs{
		Host:       "localhost",
		HostPort:   5672,
		User:       "admin",
		Password:   "s3cr3t",
		Database:   "/",
		ExtraPorts: map[string]int{"mgmt_host_port": 15672},
	})

	want := "amqp://admin:s3cr3t@localhost:5672/%2F"
	if ci.Primary != want {
		t.Errorf("Primary = %q, want %q", ci.Primary, want)
	}
	if !strings.Contains(ci.MaskedPrimary, "****") {
		t.Errorf("MaskedPrimary should contain ****, got %q", ci.MaskedPrimary)
	}
	if strings.Contains(ci.MaskedPrimary, "s3cr3t") {
		t.Errorf("MaskedPrimary must not contain the password")
	}
	if len(ci.Endpoints) == 0 {
		t.Fatal("expected at least one Endpoint")
	}
	if ci.Endpoints[0].Label != "Management UI" {
		t.Errorf("Endpoints[0].Label = %q, want %q", ci.Endpoints[0].Label, "Management UI")
	}
	if strings.Contains(ci.Endpoints[0].Value, "s3cr3t") {
		t.Errorf("Endpoint value must not contain the password")
	}
	if !strings.Contains(ci.Endpoints[0].Value, "15672") {
		t.Errorf("Endpoint value should mention the mgmt port 15672, got %q", ci.Endpoints[0].Value)
	}
}

func TestRabbitMQConnectionInfoNoExtraPort(t *testing.T) {
	r := &rabbitmq{}
	ci := r.ConnectionInfo(ConnArgs{
		Host:     "localhost",
		HostPort: 5672,
		User:     "admin",
		Password: "pass",
		Database: "/",
		// ExtraPorts absent — Endpoints should be empty
	})
	if len(ci.Endpoints) != 0 {
		t.Errorf("expected no Endpoints when ExtraPorts is absent, got %d", len(ci.Endpoints))
	}
}

func TestRabbitMQConnectionInfoVhostEncoding(t *testing.T) {
	r := &rabbitmq{}
	tests := []struct {
		vhost   string
		wantSub string // substring expected in Primary
	}{
		{"/", "%2F"},       // default vhost must be URL-escaped
		{"myapp", "myapp"}, // plain vhost passes through
		{"my vhost/staging", "my%20vhost%2Fstaging"}, // spaces and slashes encoded
	}
	for _, tc := range tests {
		ci := r.ConnectionInfo(ConnArgs{
			Host: "h", HostPort: 1, User: "u", Password: "p", Database: tc.vhost,
		})
		if !strings.Contains(ci.Primary, tc.wantSub) {
			t.Errorf("vhost %q: Primary does not contain %q; got %s", tc.vhost, tc.wantSub, ci.Primary)
		}
	}
}

func TestRabbitMQConnectionInfoEmptyVhostDefaultsToRoot(t *testing.T) {
	r := &rabbitmq{}
	ci := r.ConnectionInfo(ConnArgs{
		Host: "h", HostPort: 1, User: "u", Password: "p", Database: "",
	})
	if !strings.Contains(ci.Primary, "%2F") {
		t.Errorf("empty vhost should default to '/' (%%2F), got %q", ci.Primary)
	}
}

func TestRabbitMQConnectionInfoMaskingWithAtInPassword(t *testing.T) {
	r := &rabbitmq{}
	ci := r.ConnectionInfo(ConnArgs{
		Host: "localhost", HostPort: 5672, User: "u", Password: "p@ss", Database: "/",
	})
	if strings.Contains(ci.MaskedPrimary, "p@ss") {
		t.Errorf("MaskedPrimary must not contain the password, got %q", ci.MaskedPrimary)
	}
	if !strings.Contains(ci.MaskedPrimary, "****") {
		t.Errorf("MaskedPrimary must contain ****, got %q", ci.MaskedPrimary)
	}
}

func TestRabbitMQEnvVars(t *testing.T) {
	r := &rabbitmq{}

	ev := r.EnvVars("admin", "pass", "/")
	if ev["RABBITMQ_DEFAULT_USER"] != "admin" {
		t.Errorf("expected RABBITMQ_DEFAULT_USER=admin, got %q", ev["RABBITMQ_DEFAULT_USER"])
	}
	if ev["RABBITMQ_DEFAULT_VHOST"] != "/" {
		t.Errorf("expected RABBITMQ_DEFAULT_VHOST=/, got %q", ev["RABBITMQ_DEFAULT_VHOST"])
	}

	// Empty vhost must default to "/"
	ev2 := r.EnvVars("u", "p", "")
	if ev2["RABBITMQ_DEFAULT_VHOST"] != "/" {
		t.Errorf("empty vhost: expected RABBITMQ_DEFAULT_VHOST=/, got %q", ev2["RABBITMQ_DEFAULT_VHOST"])
	}
}

func TestRabbitMQExtraPortsGating(t *testing.T) {
	r := &rabbitmq{}

	// Management image → ExtraPorts returned
	extras := r.ExtraPorts("rabbitmq:3-management-alpine", nil)
	if len(extras) != 1 {
		t.Fatalf("expected 1 ExtraPort for management image, got %d", len(extras))
	}
	if extras[0].ContainerPort != 15672 {
		t.Errorf("ContainerPort = %d, want 15672", extras[0].ContainerPort)
	}
	if extras[0].OptionKey != "mgmt_host_port" {
		t.Errorf("OptionKey = %q, want %q", extras[0].OptionKey, "mgmt_host_port")
	}

	// Non-management image → nil
	if r.ExtraPorts("rabbitmq:3-alpine", nil) != nil {
		t.Error("expected nil ExtraPorts for non-management image")
	}
	if r.ExtraPorts("rabbitmq:4", nil) != nil {
		t.Error("expected nil ExtraPorts for plain version tag")
	}
}

func TestRabbitMQWizardPromptsGating(t *testing.T) {
	r := &rabbitmq{}

	if r.WizardPrompts("rabbitmq:3-management-alpine") == nil {
		t.Error("expected WizardPrompts for management image")
	}
	if r.WizardPrompts("rabbitmq:3-alpine") != nil {
		t.Error("expected nil WizardPrompts for non-management image")
	}
}

func TestRabbitMQValidatePassword(t *testing.T) {
	r := &rabbitmq{}
	if err := r.ValidatePassword(""); err == nil {
		t.Error("expected error for empty password")
	}
	if err := r.ValidatePassword("anypass"); err != nil {
		t.Errorf("unexpected error for non-empty password: %v", err)
	}
}

func TestOptionalPortValidator(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"", false},
		{"15672", false},
		{"1", false},
		{"65535", false},
		{"0", true},
		{"-1", true},
		{"65536", true},
		{"abc", true},
	}
	for _, tc := range tests {
		err := optionalPortValidator(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("optionalPortValidator(%q): err=%v, wantErr=%v", tc.input, err, tc.wantErr)
		}
	}
}
