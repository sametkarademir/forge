package commands

import (
	"fmt"
	"os"

	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

func NewConnCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "conn <project>",
		Short: "Print the connection string for a project (suitable for piping)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn, err := service.GetConnectionString(cmd.Context(), args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, "✗ "+err.Error())
				os.Exit(1)
			}
			fmt.Println(dsn)
			return nil
		},
	}
}
