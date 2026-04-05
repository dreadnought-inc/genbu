package resolver

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/expr"
	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/validator"
)

var refPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// Resolver resolves environment variables from config using providers.
type Resolver struct {
	registry *provider.Registry
}

// New creates a new Resolver with the given provider registry.
func New(registry *provider.Registry) *Resolver {
	return &Resolver{registry: registry}
}

// Result holds the resolved variables and config defaults for validation.
type Result struct {
	Vars     []validator.Input
	Defaults *config.Defaults
}

// Resolve processes the config and resolves all variable values.
func (r *Resolver) Resolve(ctx context.Context, cfg *config.Config) (*Result, error) {
	flatVars := cfg.Flatten()

	// Phase 1: resolve raw values from providers/literals/env
	rawValues := make(map[string]string, len(flatVars))
	varConfigs := make(map[string]config.Variable, len(flatVars))

	for _, v := range flatVars {
		value, err := r.resolveVar(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", v.Name, err)
		}

		if value == "" && v.Default != "" {
			value = v.Default
		}

		rawValues[v.Name] = value
		varConfigs[v.Name] = v
	}

	// Phase 2: build dependency graph from ${VAR} and ${{ expr }} references
	deps := make(map[string][]string, len(flatVars))
	for _, v := range flatVars {
		refs := extractAllRefs(rawValues[v.Name])
		deps[v.Name] = refs
	}

	order, err := topoSort(flatVars, deps)
	if err != nil {
		return nil, err
	}

	// Phase 3: expand ${VAR} references, then evaluate ${{ expr }} expressions
	resolved := make(map[string]string, len(flatVars))
	for _, name := range order {
		value := expandRefs(rawValues[name], resolved)

		value, err := expr.Eval(value, resolved)
		if err != nil {
			return nil, fmt.Errorf("evaluating expressions in %s: %w", name, err)
		}

		resolved[name] = value
	}

	// Phase 3: build result preserving original order
	inputs := make([]validator.Input, 0, len(flatVars))
	for _, v := range flatVars {
		inputs = append(inputs, validator.Input{
			Name:     v.Name,
			Value:    resolved[v.Name],
			Validate: v.Validate,
		})
	}

	return &Result{
		Vars:     inputs,
		Defaults: cfg.Defaults,
	}, nil
}

// extractAllRefs returns variable names referenced via ${VAR} and ${{ expr }} in the value.
func extractAllRefs(value string) []string {
	seen := make(map[string]bool)
	var refs []string

	// Collect ${VAR} references
	for _, m := range refPattern.FindAllStringSubmatch(value, -1) {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			refs = append(refs, name)
		}
	}

	// Collect variable references inside ${{ expr }} expressions
	for _, ref := range expr.ExtractExprVarRefs(value) {
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	return refs
}

// expandRefs replaces ${VAR} references in value with resolved values.
func expandRefs(value string, resolved map[string]string) string {
	return refPattern.ReplaceAllStringFunc(value, func(match string) string {
		name := match[2 : len(match)-1] // strip ${ and }
		if v, ok := resolved[name]; ok {
			return v
		}
		return match // keep unresolved references as-is
	})
}

// topoSort performs topological sort with circular reference detection.
func topoSort(vars []config.Variable, deps map[string][]string) ([]string, error) {
	// Filter deps to only include references to known variables
	known := make(map[string]bool, len(vars))
	for _, v := range vars {
		known[v.Name] = true
	}

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)

	state := make(map[string]int, len(vars))
	order := make([]string, 0, len(vars))
	var path []string

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case visited:
			return nil
		case visiting:
			cycle := buildCyclePath(path, name)
			return fmt.Errorf("circular reference detected: %s", strings.Join(cycle, " -> "))
		}

		state[name] = visiting
		path = append(path, name)

		for _, dep := range deps[name] {
			if !known[dep] {
				continue // reference to external/unknown var, skip
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		state[name] = visited
		order = append(order, name)
		return nil
	}

	for _, v := range vars {
		if err := visit(v.Name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// buildCyclePath extracts the cycle portion from the DFS path.
func buildCyclePath(path []string, target string) []string {
	for i, name := range path {
		if name == target {
			cycle := make([]string, len(path)-i+1)
			copy(cycle, path[i:])
			cycle[len(cycle)-1] = target
			return cycle
		}
	}
	return []string{target, target}
}

func (r *Resolver) resolveVar(ctx context.Context, v config.Variable) (string, error) {
	// Literal value specified directly
	if v.Value != "" && v.Source == nil {
		return v.Value, nil
	}

	// No source and no value: read from existing env
	if v.Source == nil {
		return os.Getenv(v.Name), nil
	}

	// Source type "env": read the variable's own name from env
	if v.Source.Type == "env" {
		src := &config.SourceConfig{
			Type: "env",
			Path: v.Name,
		}
		p, err := r.registry.Get("env")
		if err != nil {
			return "", err
		}
		return p.Resolve(ctx, src)
	}

	// Use the registered provider for the source type
	p, err := r.registry.Get(v.Source.Type)
	if err != nil {
		return "", err
	}

	return p.Resolve(ctx, v.Source)
}
