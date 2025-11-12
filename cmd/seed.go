package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"go-base/internal/app"
	"go-base/internal/seed"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with initial data",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return errors.New("configuration not initialized")
		}

		application, err := app.New(rootCtx, cfg)
		if err != nil {
			return err
		}
		defer application.Close(context.Background())

		return seed.Run(rootCtx, cfg, application.DB, application.Cache, application.PasswordSvc)
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
