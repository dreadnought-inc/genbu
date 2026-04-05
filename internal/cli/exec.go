package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/executor"
	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/resolver"
	"github.com/dreadnought-inc/genbu/internal/validator"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec -- COMMAND [ARGS...]",
		Short: "Resolve variables, validate, and exec a command",
		Long:  "Resolves environment variables from config, validates them, sets them in the environment, and replaces the current process with the specified command.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runExec,
	}
}

func runExec(_ *cobra.Command, args []string) error {
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

	envVars := make(map[string]string, len(result.Vars))
	for _, v := range result.Vars {
		envVars[v.Name] = v.Value
	}

	return executor.Exec(args, envVars)
}

func init() {
	rootCmd.AddCommand(newExecCmd())
}
