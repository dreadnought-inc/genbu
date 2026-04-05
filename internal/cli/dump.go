package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/resolver"
	"github.com/dreadnought-inc/genbu/internal/validator"
)

var (
	dumpFormat string
	dumpMask   bool
)

var dumpCmd *cobra.Command

func newDumpCmd() *cobra.Command {
	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Resolve and print variables for debugging",
		Long:  "Resolves environment variables from config and prints them. Use --mask to redact sensitive values.",
		RunE:  runDump,
	}

	dumpCmd.Flags().StringVar(&dumpFormat, "format", "dotenv", "output format: dotenv, ini, toml, json")
	dumpCmd.Flags().BoolVar(&dumpMask, "mask", false, "mask values (show first/last 2 chars)")

	return dumpCmd
}

// resolveDumpFormat returns the effective dump format using the priority:
// 1. --format CLI flag (if explicitly set)
// 2. dump_format in genbu.yaml
// 3. "dotenv" (default)
func resolveDumpFormat(cfg *config.Config) string {
	if dumpCmd.Flags().Changed("format") {
		return dumpFormat
	}
	if cfg.DumpFormat != "" {
		return cfg.DumpFormat
	}
	return "dotenv"
}

func runDump(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadFile(configFile)
	if err != nil {
		return err
	}

	format := resolveDumpFormat(cfg)

	registry := provider.NewDefaultRegistry()
	registerAWSProviders(registry)

	r := resolver.New(registry)
	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("resolving variables: %w", err)
	}

	vars := applyMask(result.Vars)

	switch format {
	case "dotenv", "env":
		return writeDotenv(os.Stdout, vars)
	case "ini":
		return writeINI(os.Stdout, vars)
	case "toml":
		return writeTOML(os.Stdout, vars)
	case "json":
		return writeJSON(os.Stdout, vars)
	default:
		return fmt.Errorf("unsupported format: %s (supported: dotenv, ini, toml, json)", dumpFormat)
	}
}

type kv struct {
	Name  string
	Value string
}

func applyMask(inputs []validator.Input) []kv {
	vars := make([]kv, len(inputs))
	for i, v := range inputs {
		vars[i] = kv{Name: v.Name, Value: maskValue(v.Value)}
	}
	return vars
}

func writeDotenv(w io.Writer, vars []kv) error {
	for _, v := range vars {
		var err error
		if strings.ContainsAny(v.Value, " \t\"'#$\\") || v.Value == "" {
			_, err = fmt.Fprintf(w, "%s=%q\n", v.Name, v.Value)
		} else {
			_, err = fmt.Fprintf(w, "%s=%s\n", v.Name, v.Value)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeINI(w io.Writer, vars []kv) error {
	sections := make(map[string][]kv)
	var noSection []kv
	var order []string

	for _, v := range vars {
		parts := splitEnvName(v.Name)
		if len(parts) >= 2 {
			section := strings.ToLower(parts[0])
			key := strings.ToLower(strings.Join(parts[1:], "_"))
			if _, exists := sections[section]; !exists {
				order = append(order, section)
			}
			sections[section] = append(sections[section], kv{Name: key, Value: v.Value})
		} else {
			noSection = append(noSection, kv{Name: strings.ToLower(v.Name), Value: v.Value})
		}
	}

	for _, v := range noSection {
		if _, err := fmt.Fprintf(w, "%s = %s\n", v.Name, v.Value); err != nil {
			return err
		}
	}
	if len(noSection) > 0 && len(order) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	for i, section := range order {
		if _, err := fmt.Fprintf(w, "[%s]\n", section); err != nil {
			return err
		}
		for _, v := range sections[section] {
			if _, err := fmt.Fprintf(w, "%s = %s\n", v.Name, v.Value); err != nil {
				return err
			}
		}
		if i < len(order)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeTOML(w io.Writer, vars []kv) error {
	sections := make(map[string]map[string]string)
	var topLevel []kv
	var order []string

	for _, v := range vars {
		parts := splitEnvName(v.Name)
		if len(parts) >= 2 {
			section := strings.ToLower(parts[0])
			key := strings.ToLower(strings.Join(parts[1:], "_"))
			if _, exists := sections[section]; !exists {
				sections[section] = make(map[string]string)
				order = append(order, section)
			}
			sections[section][key] = v.Value
		} else {
			topLevel = append(topLevel, kv{Name: strings.ToLower(v.Name), Value: v.Value})
		}
	}

	for _, v := range topLevel {
		if err := encodeTOMLValue(w, v.Name, v.Value); err != nil {
			return err
		}
	}
	if len(topLevel) > 0 && len(order) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	for i, section := range order {
		if _, err := fmt.Fprintf(w, "[%s]\n", section); err != nil {
			return err
		}
		for key, val := range sections[section] {
			if err := encodeTOMLValue(w, key, val); err != nil {
				return err
			}
		}
		if i < len(order)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeTOMLValue(w io.Writer, key, value string) error {
	m := map[string]string{key: value}
	if err := toml.NewEncoder(w).Encode(m); err != nil {
		_, fmtErr := fmt.Fprintf(w, "%s = %q\n", key, value)
		return fmtErr
	}
	return nil
}

func writeJSON(w io.Writer, vars []kv) error {
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Name] = v.Value
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// splitEnvName splits an env var name by underscore, treating the first
// segment as a potential section name if there are multiple segments.
func splitEnvName(name string) []string {
	return strings.SplitN(name, "_", 2)
}

func maskValue(value string) string {
	if !dumpMask {
		return value
	}

	if len(value) <= 4 {
		return "****"
	}

	return value[:2] + "****" + value[len(value)-2:]
}

func init() {
	rootCmd.AddCommand(newDumpCmd())
}
