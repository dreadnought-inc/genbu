package provider

import (
	"context"
	"fmt"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// Provider resolves a single environment variable value from an external source.
type Provider interface {
	// Type returns the source type string this provider handles (e.g., "aws-ssm").
	Type() string

	// Resolve fetches the value for the given source configuration.
	Resolve(ctx context.Context, src *config.SourceConfig) (string, error)
}

// Registry holds registered providers keyed by source type.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Type()] = p
}

// Get returns the provider for the given source type.
func (r *Registry) Get(sourceType string) (Provider, error) {
	p, ok := r.providers[sourceType]
	if !ok {
		return nil, fmt.Errorf("unknown provider type: %q", sourceType)
	}
	return p, nil
}
