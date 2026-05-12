package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sametkarademir/forge/internal/build"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/updater"
)

func NewUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update forge to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if build.Version == "dev" {
				logger.Warn("Running in dev mode — update is disabled")
				return nil
			}

			logger.Info("Checking for updates...")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			newVer, err := updater.Apply(ctx)
			if err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			if newVer == "" {
				logger.Success(fmt.Sprintf("Already up to date (%s)", build.Version))
				return nil
			}
			logger.Success(fmt.Sprintf("Updated to %s — restart your terminal to use the new version", newVer))
			return nil
		},
	}
}
