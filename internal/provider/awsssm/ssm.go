package awsssm

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// SSMClient is the subset of the SSM API used by this provider.
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// Provider fetches values from AWS SSM Parameter Store.
type Provider struct {
	client SSMClient
}

// New creates a Provider with the given SSM client.
func New(client SSMClient) *Provider {
	return &Provider{client: client}
}

// NewFromConfig creates a Provider using the default AWS config.
func NewFromConfig(ctx context.Context) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}
	return &Provider{client: ssm.NewFromConfig(cfg)}, nil
}

// Type returns "aws-ssm".
func (p *Provider) Type() string {
	return "aws-ssm"
}

// Resolve fetches a parameter value from SSM Parameter Store.
func (p *Provider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	if src.Path == "" {
		return "", fmt.Errorf("aws-ssm: path is required")
	}

	input := &ssm.GetParameterInput{
		Name:           &src.Path,
		WithDecryption: boolPtr(true),
	}

	var opts []func(*ssm.Options)
	if src.Region != "" {
		opts = append(opts, func(o *ssm.Options) {
			o.Region = src.Region
		})
	}

	output, err := p.client.GetParameter(ctx, input, opts...)
	if err != nil {
		return "", fmt.Errorf("aws-ssm: getting parameter %q: %w", src.Path, err)
	}

	if output.Parameter == nil || output.Parameter.Value == nil {
		return "", fmt.Errorf("aws-ssm: parameter %q has no value", src.Path)
	}

	return *output.Parameter.Value, nil
}

func boolPtr(b bool) *bool { return &b }
