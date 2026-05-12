package commands

import (
	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
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
			if user == "" {
				user = config.DefaultUser()
			}
			if password == "" {
				password = config.DefaultPassword()
			}
			if db == "" {
				db = config.DefaultDB()
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
	_ = cmd.MarkFlagRequired("engine")
	cmd.Flags().StringVar(&image, "image", "", "Override Docker image tag")
	cmd.Flags().StringVar(&user, "user", "", "Database username (default: config)")
	cmd.Flags().StringVar(&password, "password", "", "Database password (default: config)")
	cmd.Flags().StringVar(&db, "db", "", "Database name (default: config)")
	cmd.Flags().IntVar(&port, "port", 0, "Host port (default: auto-allocate from config range)")

	return cmd
}
