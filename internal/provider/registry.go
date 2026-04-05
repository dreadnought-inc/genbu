package provider

import (
	"github.com/dreadnought-inc/genbu/internal/provider/env"
	"github.com/dreadnought-inc/genbu/internal/provider/literal"
)

// NewDefaultRegistry creates a registry with all built-in providers.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(&literal.Provider{})
	r.Register(&env.Provider{})
	return r
}
