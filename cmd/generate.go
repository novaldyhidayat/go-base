package cmd

import "github.com/spf13/cobra"

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Code generation utilities",
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
