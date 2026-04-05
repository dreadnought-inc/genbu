package cli

import (
	"context"
	"log"

	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/provider/awssm"
	"github.com/dreadnought-inc/genbu/internal/provider/awsssm"
)

// registerAWSProviders registers AWS providers.
// Provider initialization errors are logged but do not prevent startup,
// so configs that don't use AWS providers still work without AWS credentials.
func registerAWSProviders(registry *provider.Registry) {
	ctx := context.Background()

	ssmProvider, err := awsssm.NewFromConfig(ctx)
	if err != nil {
		log.Printf("warning: aws ssm provider unavailable: %v", err)
	} else {
		registry.Register(ssmProvider)
	}

	smProvider, err := awssm.NewFromConfig(ctx)
	if err != nil {
		log.Printf("warning: aws secrets manager provider unavailable: %v", err)
	} else {
		registry.Register(smProvider)
	}
}
