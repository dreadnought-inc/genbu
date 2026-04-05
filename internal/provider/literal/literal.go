package literal

import (
	"context"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// Provider returns the literal value specified in the variable config.
// This is used when a variable has a "value" field directly set.
type Provider struct{}

// Type returns "literal".
func (p *Provider) Type() string {
	return "literal"
}

// Resolve returns the value from the source config path field,
// which is repurposed to carry the literal value.
func (p *Provider) Resolve(_ context.Context, src *config.SourceConfig) (string, error) {
	return src.Path, nil
}
