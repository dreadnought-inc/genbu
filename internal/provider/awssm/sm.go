package awssm

import (
	"context"
	"encoding/json"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// SMClient is the subset of the Secrets Manager API used by this provider.
type SMClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// Provider fetches values from AWS Secrets Manager.
type Provider struct {
	client SMClient
}

// New creates a Provider with the given Secrets Manager client.
func New(client SMClient) *Provider {
	return &Provider{client: client}
}

// NewFromConfig creates a Provider using the default AWS config.
func NewFromConfig(ctx context.Context) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}
	return &Provider{client: secretsmanager.NewFromConfig(cfg)}, nil
}

// Type returns "aws-secretsmanager".
func (p *Provider) Type() string {
	return "aws-secretsmanager"
}

// Resolve fetches a secret value from AWS Secrets Manager.
// If json_key is specified, the secret is parsed as JSON and the specified key is extracted.
func (p *Provider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	if src.SecretID == "" {
		return "", fmt.Errorf("aws-secretsmanager: secret_id is required")
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: &src.SecretID,
	}

	var opts []func(*secretsmanager.Options)
	if src.Region != "" {
		opts = append(opts, func(o *secretsmanager.Options) {
			o.Region = src.Region
		})
	}

	output, err := p.client.GetSecretValue(ctx, input, opts...)
	if err != nil {
		return "", fmt.Errorf("aws-secretsmanager: getting secret %q: %w", src.SecretID, err)
	}

	if output.SecretString == nil {
		return "", fmt.Errorf("aws-secretsmanager: secret %q has no string value", src.SecretID)
	}

	value := *output.SecretString

	// If json_key is set, extract the key from JSON
	if src.JSONKey != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return "", fmt.Errorf("aws-secretsmanager: parsing secret %q as JSON: %w", src.SecretID, err)
		}

		v, ok := m[src.JSONKey]
		if !ok {
			return "", fmt.Errorf("aws-secretsmanager: key %q not found in secret %q", src.JSONKey, src.SecretID)
		}

		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("aws-secretsmanager: key %q in secret %q is not a string", src.JSONKey, src.SecretID)
		}

		return str, nil
	}

	return value, nil
}
