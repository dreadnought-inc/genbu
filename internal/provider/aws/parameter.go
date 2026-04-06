package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// SSMClient is the subset of the SSM API used by the parameter provider.
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// ParameterProvider fetches values from AWS SSM Parameter Store.
type ParameterProvider struct {
	client SSMClient
}

// NewParameterProvider creates a ParameterProvider with the given SSM client.
func NewParameterProvider(client SSMClient) *ParameterProvider {
	return &ParameterProvider{client: client}
}

// Type returns "parameter".
func (p *ParameterProvider) Type() string {
	return "parameter"
}

// Resolve fetches a parameter value from SSM Parameter Store.
func (p *ParameterProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("aws parameter: key is required")
	}

	withDecryption := true
	input := &ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: &withDecryption,
	}

	var opts []func(*ssm.Options)
	if src.Region != "" {
		opts = append(opts, func(o *ssm.Options) {
			o.Region = src.Region
		})
	}

	output, err := p.client.GetParameter(ctx, input, opts...)
	if err != nil {
		return "", fmt.Errorf("aws parameter: getting %q: %w", key, err)
	}

	if output.Parameter == nil || output.Parameter.Value == nil {
		return "", fmt.Errorf("aws parameter: %q has no value", key)
	}

	return *output.Parameter.Value, nil
}
