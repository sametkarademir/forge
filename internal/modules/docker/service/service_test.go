package service

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// fakeEngine satisfies engines.Engine for testing without importing the engines package.
type fakeEngine struct{ defaultImage string }

func (f *fakeEngine) Name() string                                              { return "fake" }
func (f *fakeEngine) DefaultImage() string                                      { return f.defaultImage }
func (f *fakeEngine) ImageRepos() []string                                      { return nil }
func (f *fakeEngine) DefaultPort() int                                          { return 9999 }
func (f *fakeEngine) DataDir(_ string) string                                   { return "/data" }
func (f *fakeEngine) EnvVars(u, p, db string) map[string]string                 { return nil }
func (f *fakeEngine) ConnectionString(h string, p int, u, pw, db string) string { return "" }
func (f *fakeEngine) ValidatePassword(pw string) error                          { return nil }
func (f *fakeEngine) PasswordEnvKey() string                                    { return "FAKE_PASSWORD" }

func TestResolveImage(t *testing.T) {
	eng := &fakeEngine{defaultImage: "fake:latest"}

	t.Run("flag wins over everything", func(t *testing.T) {
		viper.Set("docker.engines.fake.default_image", "fake:config-override")
		t.Cleanup(func() { viper.Set("docker.engines.fake.default_image", "") })

		got := resolveImage("fake:flag", "fake", eng)
		require.Equal(t, "fake:flag", got)
	})

	t.Run("config override wins over engine default", func(t *testing.T) {
		viper.Set("docker.engines.fake.default_image", "fake:config-override")
		t.Cleanup(func() { viper.Set("docker.engines.fake.default_image", "") })

		got := resolveImage("", "fake", eng)
		require.Equal(t, "fake:config-override", got)
	})

	t.Run("engine default used when nothing else set", func(t *testing.T) {
		viper.Set("docker.engines.fake.default_image", "")

		got := resolveImage("", "fake", eng)
		require.Equal(t, "fake:latest", got)
	})
}
