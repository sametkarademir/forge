package commands

import (
	"fmt"

	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/core/ui"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	var (
		engine   string
		image    string
		user     string
		password string
		db       string
		port     int
	)

	cmd := &cobra.Command{
		Use:   "create <project>",
		Short: "Create a managed database container for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			interactive := ui.IsInteractive()

			// Engine: required. Prompt when missing and stdin is a TTY.
			if engine == "" {
				if !interactive {
					return fmt.Errorf("required flag \"engine\" not set")
				}
				var err error
				engine, err = promptEngine()
				if err != nil {
					logger.Error(err.Error())
					return err
				}
			}

			eng, ok := engines.Get(engine)
			if !ok {
				err := engines.ErrUnknownEngine(engine)
				logger.Error(err.Error())
				return err
			}

			// Image: prompt when missing and interactive.
			if image == "" && interactive {
				var err error
				image, err = promptImage(cmd.Context(), engine)
				if err != nil {
					logger.Error(err.Error())
					return err
				}
			}

			// User
			if user == "" {
				if interactive {
					var err error
					user, err = promptUser()
					if err != nil {
						logger.Error(err.Error())
						return err
					}
				} else {
					user = config.DefaultUser()
				}
			}

			// Password
			if password == "" {
				if interactive {
					var err error
					password, err = promptPassword(eng)
					if err != nil {
						logger.Error(err.Error())
						return err
					}
				} else {
					password = config.DefaultPassword()
				}
			}

			// DB name
			if db == "" {
				if interactive {
					var err error
					db, err = promptDB()
					if err != nil {
						logger.Error(err.Error())
						return err
					}
				} else {
					db = config.DefaultDB()
				}
			}

			_, err := service.CreateProject(cmd.Context(), service.CreateOptions{
				ProjectName: args[0],
				Engine:      engine,
				Image:       image,
				User:        user,
				Password:    password,
				Database:    db,
				HostPort:    port,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&engine, "engine", "e", "", "Database engine (postgres, mssql, mysql)")
	cmd.Flags().StringVar(&image, "image", "", "Override Docker image tag")
	cmd.Flags().StringVar(&user, "user", "", "Database username (default: config)")
	cmd.Flags().StringVar(&password, "password", "", "Database password (default: config)")
	cmd.Flags().StringVar(&db, "db", "", "Database name (default: config)")
	cmd.Flags().IntVar(&port, "port", 0, "Host port (default: auto-allocate from config range)")

	return cmd
}
