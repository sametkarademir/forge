package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sametkarademir/forge/internal/build"
	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/registry"
	"github.com/sametkarademir/forge/internal/updater"

	_ "github.com/sametkarademir/forge/internal/modules/docker"
	_ "github.com/sametkarademir/forge/internal/modules/update"
)

func main() {
	updateNotifCh := make(chan string, 1)
	if isTerminal() && build.Version != "dev" && updater.ShouldCheck() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			updater.RecordCheck()
			if newVer, found := updater.CheckLatest(ctx); found {
				updateNotifCh <- newVer
			}
			close(updateNotifCh)
		}()
	} else {
		close(updateNotifCh)
	}

	root := &cobra.Command{
		Use:     "forge",
		Short:   "Developer productivity CLI",
		Version: build.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.Init()
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			select {
			case newVer, ok := <-updateNotifCh:
				if ok && newVer != "" {
					logger.Info(fmt.Sprintf("A new version is available: %s → %s", build.Version, newVer))
					logger.Info("Run: forge update")
				}
			default:
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetVersionTemplate(fmt.Sprintf(
		"forge {{.Version}}\ncommit: %s\nbuilt:  %s\n",
		build.Commit,
		build.Date,
	))

	for _, cmd := range registry.Commands() {
		root.AddCommand(cmd)
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "✗ "+err.Error())
		os.Exit(1)
	}
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
