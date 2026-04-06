package aws

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Config holds initialized AWS SDK clients shared by parameter and secret providers.
type Config struct {
	SSM SSMClient
	SM  SMClient
}

// NewDefaultConfig creates AWS clients from the default AWS SDK config.
func NewDefaultConfig(ctx context.Context) (*Config, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}
	return &Config{
		SSM: ssm.NewFromConfig(cfg),
		SM:  secretsmanager.NewFromConfig(cfg),
	}, nil
}
