package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// SMClient is the subset of the Secrets Manager API used by the secret provider.
type SMClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// SecretProvider fetches values from AWS Secrets Manager.
type SecretProvider struct {
	client SMClient
}

// NewSecretProvider creates a SecretProvider with the given Secrets Manager client.
func NewSecretProvider(client SMClient) *SecretProvider {
	return &SecretProvider{client: client}
}

// Type returns "secret".
func (p *SecretProvider) Type() string {
	return "secret"
}

// Resolve fetches a secret value from AWS Secrets Manager.
// If json_key is specified, the secret is parsed as JSON and the specified key is extracted.
func (p *SecretProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("aws secret: key is required")
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &key,
	}

	var opts []func(*secretsmanager.Options)
	if src.Region != "" {
		opts = append(opts, func(o *secretsmanager.Options) {
			o.Region = src.Region
		})
	}

	output, err := p.client.GetSecretValue(ctx, input, opts...)
	if err != nil {
		return "", fmt.Errorf("aws secret: getting %q: %w", key, err)
	}

	if output.SecretString == nil {
		return "", fmt.Errorf("aws secret: %q has no string value", key)
	}

	value := *output.SecretString

	if src.JSONKey != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return "", fmt.Errorf("aws secret: parsing %q as JSON: %w", key, err)
		}

		v, ok := m[src.JSONKey]
		if !ok {
			return "", fmt.Errorf("aws secret: json_key %q not found in %q", src.JSONKey, key)
		}

		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("aws secret: json_key %q in %q is not a string", src.JSONKey, key)
		}

		return str, nil
	}

	return value, nil
}
