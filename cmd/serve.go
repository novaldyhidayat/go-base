package cmd

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"go-base/internal/app"
	"go-base/internal/httpserver"
	"go-base/internal/user"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return errors.New("configuration not initialized")
		}

		application, err := app.New(rootCtx, cfg)
		if err != nil {
			return err
		}
		defer application.Close(context.Background())

		if cfg.Database.AutoMigrate {
			if err := application.DB.DB().AutoMigrate(&user.User{}); err != nil {
				return err
			}
		}

		srv := httpserver.NewServer(application.Router, cfg.Server, application.Logger)

		signalCtx, stop := signal.NotifyContext(rootCtx, syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		return srv.Run(signalCtx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
