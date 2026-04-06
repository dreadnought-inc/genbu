package env

import (
	"context"
	"os"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// Provider reads values from existing environment variables.
type Provider struct{}

// Type returns "env".
func (p *Provider) Type() string {
	return "env"
}

// Resolve reads the environment variable with the name stored in src.Key.
func (p *Provider) Resolve(_ context.Context, src *config.SourceConfig) (string, error) {
	return os.Getenv(src.EffectiveKey()), nil
}
