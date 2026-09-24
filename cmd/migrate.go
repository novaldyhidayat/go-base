package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"go-base/internal/database"
	"go-base/internal/database/migrations"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return errors.New("configuration not initialized")
		}

		db, err := database.NewPostgres(rootCtx, cfg.Database)
		if err != nil {
			return err
		}
		defer db.Close()

		return migrations.AutoMigrate(db.DB())
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
