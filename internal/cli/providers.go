package cli

import (
	"context"
	"fmt"
	"log"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
	"github.com/dreadnought-inc/genbu/internal/provider/aws"
	"github.com/dreadnought-inc/genbu/internal/provider/azure"
	"github.com/dreadnought-inc/genbu/internal/provider/gcp"
)

// resolveProvider determines the effective provider using priority:
// 1. --provider CLI flag
// 2. provider in config YAML
// 3. "aws" (default)
func resolveProvider(cfg *config.Config) string {
	if providerFlag != "" {
		return providerFlag
	}
	if cfg.Provider != "" {
		return cfg.Provider
	}
	return "aws"
}

// registerCloudProviders registers cloud-specific providers based on the provider name.
func registerCloudProviders(registry *provider.Registry, providerName string) error {
	switch providerName {
	case "aws":
		return registerAWSProviders(registry)
	case "gcp":
		return registerGCPProviders(registry)
	case "azure":
		return registerAzureProviders(registry)
	default:
		return fmt.Errorf("unknown provider: %s", providerName)
	}
}

func registerAWSProviders(registry *provider.Registry) error {
	ctx := context.Background()

	cfg, err := aws.NewDefaultConfig(ctx)
	if err != nil {
		log.Printf("warning: aws providers unavailable: %v", err)
		return nil
	}

	registry.Register(aws.NewParameterProvider(cfg.SSM))
	registry.Register(aws.NewSecretProvider(cfg.SM))
	return nil
}

func registerGCPProviders(registry *provider.Registry) error {
	ctx := context.Background()

	cfg, err := gcp.NewDefaultConfig(ctx)
	if err != nil {
		log.Printf("warning: gcp providers unavailable: %v", err)
		return nil
	}

	registry.Register(gcp.NewParameterProvider(cfg.SM))
	registry.Register(gcp.NewSecretProvider(cfg.SM))
	return nil
}

func registerAzureProviders(registry *provider.Registry) error {
	ctx := context.Background()

	cfg, err := azure.NewDefaultConfig(ctx)
	if err != nil {
		log.Printf("warning: azure providers unavailable: %v", err)
		return nil
	}

	if cfg.AppConfig != nil {
		registry.Register(azure.NewParameterProvider(cfg.AppConfig))
	}
	if cfg.KeyVault != nil {
		registry.Register(azure.NewSecretProvider(cfg.KeyVault))
	}
	return nil
}
