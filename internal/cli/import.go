package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dreadnought-inc/genbu/internal/importer"
)

var importFormat string

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import FILE",
		Short: "Generate genbu.yaml template from an existing config file",
		Long:  "Reads a dotenv, INI, or TOML file and outputs a genbu.yaml template to stdout.",
		Args:  cobra.ExactArgs(1),
		RunE:  runImport,
	}

	cmd.Flags().StringVarP(&importFormat, "format", "f", "", "source format: dotenv, ini, toml (auto-detected from extension if omitted)")

	return cmd
}

func runImport(_ *cobra.Command, args []string) error {
	path := args[0]

	format := importFormat
	if format == "" {
		format = importer.FormatFromExtension(path)
		if format == "" {
			return fmt.Errorf("cannot detect format from %q, use --format to specify", path)
		}
	}

	imp, err := importer.Get(format)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	vars, err := imp.Import(f)
	if err != nil {
		return fmt.Errorf("importing %s: %w", path, err)
	}

	out, err := importer.GenerateYAML(vars)
	if err != nil {
		return fmt.Errorf("generating yaml: %w", err)
	}

	fmt.Print(string(out))
	return nil
}

func init() {
	rootCmd.AddCommand(newImportCmd())
}
