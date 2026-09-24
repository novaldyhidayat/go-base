package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"go-base/internal/config"
	"go-base/internal/logger"
)

var (
	cfg        *config.Config
	rootCtx    context.Context
	rootCancel context.CancelFunc
)

var rootCmd = &cobra.Command{
	Use:   "go-base",
	Short: "Go Base is a production-ready starter kit for Go services",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil {
			return nil
		}

		configPath, _ := cmd.Flags().GetString("config")

		c, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cmd.Flags().Changed("env") {
			env, err := cmd.Flags().GetString("env")
			if err != nil {
				return fmt.Errorf("read environment flag: %w", err)
			}
			c.App.Env = env
		}

		logger.SetGlobalLogger(c.Logging)

		rootCtx, rootCancel = context.WithCancel(context.Background())

		cfg = c

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if rootCancel != nil {
			rootCancel()
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("config", "c", "configs/config.yaml", "configuration file path")
	rootCmd.PersistentFlags().String("env", "development", "application environment")

	_ = viper.BindPFlag("app.env", rootCmd.PersistentFlags().Lookup("env"))
}

func initConfig() {
	viper.SetEnvPrefix("GOBASE")
	viper.AutomaticEnv()
}

// Execute launches the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// AppConfig returns the loaded configuration.
func AppConfig() *config.Config {
	return cfg
}

// RootContext returns the context derived during command execution.
func RootContext() context.Context {
	return rootCtx
}
