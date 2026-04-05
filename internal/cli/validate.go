package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/resolver"
	"github.com/dreadnought-inc/genbu/internal/validator"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Resolve and validate variables without executing a command",
		Long:  "Resolves environment variables from config and validates them. Exits with code 0 if all validations pass, or 1 if any fail. Useful in CI pipelines.",
		RunE:  runValidate,
	}
}

func runValidate(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadFile(configFile)
	if err != nil {
		return err
	}

	registry := provider.NewDefaultRegistry()
	registerAWSProviders(registry)

	r := resolver.New(registry)
	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("resolving variables: %w", err)
	}

	if err := validator.Validate(result.Vars, result.Defaults); err != nil {
		fmt.Fprintln(os.Stderr, "validation failed:")
		fmt.Fprintln(os.Stderr, err)
		return fmt.Errorf("validation failed")
	}

	fmt.Println("all validations passed")
	return nil
}

func init() {
	rootCmd.AddCommand(newValidateCmd())
}
