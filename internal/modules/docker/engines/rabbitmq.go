package engines

import (
	"fmt"
	"net/url"
	"strings"
)

type rabbitmq struct{}

func init() { Register(&rabbitmq{}) }

func (r *rabbitmq) Name() string            { return "rabbitmq" }
func (r *rabbitmq) DefaultImage() string    { return "rabbitmq:3-management-alpine" }
func (r *rabbitmq) ImageRepos() []string    { return []string{"rabbitmq"} }
func (r *rabbitmq) DefaultPort() int        { return 5672 }
func (r *rabbitmq) DataDir(_ string) string { return "/var/lib/rabbitmq" }
func (r *rabbitmq) PasswordEnvKey() string  { return "RABBITMQ_DEFAULT_PASS" }

func (r *rabbitmq) EnvVars(user, password, vhost string) map[string]string {
	if vhost == "" {
		vhost = "/"
	}
	return map[string]string{
		"RABBITMQ_DEFAULT_USER":  user,
		"RABBITMQ_DEFAULT_PASS":  password,
		"RABBITMQ_DEFAULT_VHOST": vhost,
	}
}

// Cmd returns nil — the image's default ENTRYPOINT runs rabbitmq-server.
func (r *rabbitmq) Cmd(_ string) []string { return nil }

// ConnectionInfo builds an AMQP URL. The vhost is URL-path-escaped so "/" (the
// default RabbitMQ vhost) round-trips cleanly via the standard client libraries.
func (r *rabbitmq) ConnectionInfo(a ConnArgs) ConnInfo {
	vhost := a.Database
	if vhost == "" {
		vhost = "/"
	}
	raw := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		a.User, a.Password, a.Host, a.HostPort, url.PathEscape(vhost))
	masked := strings.Replace(raw, ":"+a.Password+"@", ":****@", 1)

	info := ConnInfo{Primary: raw, MaskedPrimary: masked}
	if mgmt, ok := a.ExtraPorts["mgmt_host_port"]; ok && mgmt > 0 {
		// Password intentionally omitted per Endpoint contract (no secrets in values).
		info.Endpoints = []Endpoint{{
			Label: "Management UI",
			Value: fmt.Sprintf("http://%s:%d  (login as %s)", a.Host, mgmt, a.User),
		}}
	}
	return info
}

func (r *rabbitmq) ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

// ExtraPorts returns the Management UI port binding for management-variant images.
// Non-management images (e.g. rabbitmq:3-alpine) do not expose 15672, so nil is returned.
func (r *rabbitmq) ExtraPorts(image string, _ map[string]string) []ExtraPort {
	if !strings.Contains(image, "management") {
		return nil
	}
	return []ExtraPort{
		{Label: "Management UI", ContainerPort: 15672, OptionKey: "mgmt_host_port"},
	}
}

// WizardPrompts asks for the Management UI host port when using a management image.
func (r *rabbitmq) WizardPrompts(image string) []OptionPrompt {
	if !strings.Contains(image, "management") {
		return nil
	}
	return []OptionPrompt{{
		Key:      "mgmt_host_port",
		Label:    "Management UI host port (leave empty for auto)",
		Default:  "",
		Validate: optionalPortValidator,
	}}
}

// optionalPortValidator allows an empty string (auto-assign) or a valid TCP port number.
func optionalPortValidator(s string) error {
	if s == "" {
		return nil
	}
	var n int
	if _, err := fmt.Sscan(s, &n); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("must be a port number between 1 and 65535, or leave empty for auto")
	}
	return nil
}
