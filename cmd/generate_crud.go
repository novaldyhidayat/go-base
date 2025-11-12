package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-base/internal/generate/crud"
)

var (
	crudModelPath  string
	crudStructName string
)

var crudCmd = &cobra.Command{
	Use:   "crud",
	Short: "Generate CRUD scaffolding from an existing GORM model",
	RunE: func(cmd *cobra.Command, args []string) error {
		if crudModelPath == "" {
			return fmt.Errorf("model path is required (use --model)")
		}

		opts := crud.Options{
			ModelPath:  crudModelPath,
			StructName: crudStructName,
		}

		if err := crud.Generate(opts); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "CRUD scaffolding generated for %s\n", opts.ModelPath)

		return nil
	},
}

func init() {
	crudCmd.Flags().StringVarP(&crudModelPath, "model", "m", "", "path to the GORM model file (required)")
	crudCmd.Flags().StringVarP(&crudStructName, "struct", "s", "", "struct name inside the model file (optional)")

	generateCmd.AddCommand(crudCmd)
}
