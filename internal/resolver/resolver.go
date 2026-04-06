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
	varConfigs := make(map[string]config.Variable, len(flatVars))
	for _, v := range flatVars {
		varConfigs[v.Name] = v
	}

	// Phase 1: build dependency graph.
	// Collect refs from value, default, and source.key fields.
	deps := make(map[string][]string, len(flatVars))
	for _, v := range flatVars {
		var refs []string
		refs = append(refs, extractAllRefs(v.Value)...)
		refs = append(refs, extractAllRefs(v.Default)...)
		if v.Source != nil {
			refs = append(refs, extractVarRefs(v.Source.EffectiveKey())...)
		}
		deps[v.Name] = dedupRefs(refs)
	}

	order, err := topoSort(flatVars, deps)
	if err != nil {
		return nil, err
	}

	// Phase 1.5: prefetch keys that are fully known (no ${VAR} refs in source.key).
	if prefetchErr := r.prefetchStaticKeys(ctx, flatVars); prefetchErr != nil {
		return nil, prefetchErr
	}

	// Phase 2: resolve in topological order.
	// For each variable, expand ${VAR} in source.key before calling the provider.
	resolved := make(map[string]string, len(flatVars))
	for _, name := range order {
		v := varConfigs[name]

		value, resolveErr := r.resolveVarWithExpansion(ctx, v, resolved)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolving %s: %w", name, resolveErr)
		}

		if value == "" && v.Default != "" {
			value = expandRefs(v.Default, resolved)
		}

		// Expand ${VAR} in the resolved value
		value = expandRefs(value, resolved)

		// Evaluate ${{ expr }} expressions
		value, resolveErr = expr.Eval(value, resolved)
		if resolveErr != nil {
			return nil, fmt.Errorf("evaluating expressions in %s: %w", name, resolveErr)
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

// extractVarRefs returns variable names referenced via ${VAR} in the string.
func extractVarRefs(value string) []string {
	matches := refPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}

	refs := make([]string, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, m[1])
	}
	return refs
}

// extractAllRefs returns variable names referenced via ${VAR} and ${{ expr }} in the value.
func extractAllRefs(value string) []string {
	matches := refPattern.FindAllStringSubmatch(value, -1)
	exprRefs := expr.ExtractExprVarRefs(value)
	refs := make([]string, 0, len(matches)+len(exprRefs))

	for _, m := range matches {
		refs = append(refs, m[1])
	}

	refs = append(refs, exprRefs...)
	return refs
}

// dedupRefs returns unique refs preserving order.
func dedupRefs(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(refs))
	result := make([]string, 0, len(refs))
	for _, r := range refs {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}
	return result
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
				continue
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

// prefetchStaticKeys collects source keys that are fully known (no ${VAR} refs)
// and calls Prefetch on providers that support it.
func (r *Resolver) prefetchStaticKeys(ctx context.Context, vars []config.Variable) error {
	keysByType := make(map[string][]provider.PrefetchKey)

	for _, v := range vars {
		if v.Source == nil || v.Source.Type == "" {
			continue
		}
		// Skip env and literal — they don't benefit from prefetch
		if v.Source.Type == "env" {
			continue
		}
		key := v.Source.EffectiveKey()
		if key == "" {
			continue
		}
		// Skip keys with ${VAR} references — they aren't fully known yet
		if refPattern.MatchString(key) {
			continue
		}
		keysByType[v.Source.Type] = append(keysByType[v.Source.Type], provider.PrefetchKey{
			Key:    key,
			Region: v.Source.Region,
		})
	}

	for typ, keys := range keysByType {
		p, err := r.registry.Get(typ)
		if err != nil {
			continue
		}
		if pf, ok := p.(provider.Prefetcher); ok {
			if err := pf.Prefetch(ctx, keys); err != nil {
				return fmt.Errorf("prefetching %s keys: %w", typ, err)
			}
		}
	}

	return nil
}

// resolveVarWithExpansion resolves a variable, expanding ${VAR} in source.key first.
func (r *Resolver) resolveVarWithExpansion(ctx context.Context, v config.Variable, resolved map[string]string) (string, error) {
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
			Key:  v.Name,
		}
		p, err := r.registry.Get("env")
		if err != nil {
			return "", err
		}
		return p.Resolve(ctx, src)
	}

	// Expand ${VAR} in source key before calling the provider
	src := expandSourceKey(v.Source, resolved)

	p, err := r.registry.Get(src.Type)
	if err != nil {
		return "", err
	}

	return p.Resolve(ctx, src)
}

// expandSourceKey returns a copy of the SourceConfig with ${VAR} expanded in key fields.
func expandSourceKey(src *config.SourceConfig, resolved map[string]string) *config.SourceConfig {
	expanded := *src
	expanded.Key = expandRefs(src.Key, resolved)
	expanded.Path = expandRefs(src.Path, resolved)         //nolint:staticcheck // backward compat field
	expanded.SecretID = expandRefs(src.SecretID, resolved) //nolint:staticcheck // backward compat field
	return &expanded
}
