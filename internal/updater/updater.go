package updater

import (
	"context"
	"fmt"
	"os"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/sametkarademir/forge/internal/build"
)

const repoSlug = "sametkarademir/forge"

func CheckLatest(ctx context.Context) (string, bool) {
	u, err := newUpdater()
	if err != nil {
		return "", false
	}
	release, found, err := u.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil || !found {
		return "", false
	}
	if release.GreaterThan(build.Version) {
		return release.Version(), true
	}
	return "", false
}

func Apply(ctx context.Context) (string, error) {
	u, err := newUpdater()
	if err != nil {
		return "", fmt.Errorf("creating updater: %w", err)
	}

	release, found, err := u.DetectLatest(ctx, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return "", fmt.Errorf("detecting latest release: %w", err)
	}
	if !found || !release.GreaterThan(build.Version) {
		return "", nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}

	if err := u.UpdateTo(ctx, release, exe); err != nil {
		return "", fmt.Errorf("applying update: %w", err)
	}

	return release.Version(), nil
}

func newUpdater() (*selfupdate.Updater, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{
		APIToken: githubToken(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating github source: %w", err)
	}
	return selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}

func githubToken() string {
	for _, env := range []string{"FORGE_GITHUB_TOKEN", "GITHUB_TOKEN", "HOMEBREW_GITHUB_API_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}
