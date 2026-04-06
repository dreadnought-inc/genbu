package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// AppConfigClient is the subset of the Azure App Configuration API used by this provider.
type AppConfigClient interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

// KeyVaultClient is the subset of the Azure Key Vault API used by this provider.
type KeyVaultClient interface {
	GetSecret(ctx context.Context, name string) (string, error)
}

// Config holds initialized Azure SDK clients.
type Config struct {
	AppConfig AppConfigClient
	KeyVault  KeyVaultClient
}

// NewDefaultConfig creates Azure clients using DefaultAzureCredential.
// Reads AZURE_APPCONFIG_ENDPOINT and AZURE_KEYVAULT_URL from environment.
func NewDefaultConfig(_ context.Context) (*Config, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("creating azure credential: %w", err)
	}

	cfg := &Config{}

	if endpoint := os.Getenv("AZURE_APPCONFIG_ENDPOINT"); endpoint != "" {
		client, err := azappconfig.NewClient(endpoint, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("creating azure app configuration client: %w", err)
		}
		cfg.AppConfig = &appConfigWrapper{client: client}
	}

	if kvURL := os.Getenv("AZURE_KEYVAULT_URL"); kvURL != "" {
		client, err := azsecrets.NewClient(kvURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("creating azure key vault client: %w", err)
		}
		cfg.KeyVault = &keyVaultWrapper{client: client}
	}

	return cfg, nil
}

// appConfigWrapper adapts the real Azure App Configuration client.
type appConfigWrapper struct {
	client *azappconfig.Client
}

func (w *appConfigWrapper) GetSetting(ctx context.Context, key string) (string, error) {
	resp, err := w.client.GetSetting(ctx, key, nil)
	if err != nil {
		return "", err
	}
	if resp.Value == nil {
		return "", fmt.Errorf("setting %q has no value", key)
	}
	return *resp.Value, nil
}

// keyVaultWrapper adapts the real Azure Key Vault client.
type keyVaultWrapper struct {
	client *azsecrets.Client
}

func (w *keyVaultWrapper) GetSecret(ctx context.Context, name string) (string, error) {
	resp, err := w.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", err
	}
	if resp.Value == nil {
		return "", fmt.Errorf("secret %q has no value", name)
	}
	return *resp.Value, nil
}

// ParameterProvider fetches values from Azure App Configuration.
type ParameterProvider struct {
	client AppConfigClient
}

// NewParameterProvider creates a ParameterProvider.
func NewParameterProvider(client AppConfigClient) *ParameterProvider {
	return &ParameterProvider{client: client}
}

// Type returns "parameter".
func (p *ParameterProvider) Type() string {
	return "parameter"
}

// Resolve fetches a configuration value from Azure App Configuration.
func (p *ParameterProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("azure app configuration: key is required")
	}

	value, err := p.client.GetSetting(ctx, key)
	if err != nil {
		return "", fmt.Errorf("azure app configuration: getting %q: %w", key, err)
	}

	return value, nil
}

// SecretProvider fetches values from Azure Key Vault.
type SecretProvider struct {
	client KeyVaultClient
}

// NewSecretProvider creates a SecretProvider.
func NewSecretProvider(client KeyVaultClient) *SecretProvider {
	return &SecretProvider{client: client}
}

// Type returns "secret".
func (p *SecretProvider) Type() string {
	return "secret"
}

// Resolve fetches a secret from Azure Key Vault.
func (p *SecretProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("azure key vault: key is required")
	}

	value, err := p.client.GetSecret(ctx, key)
	if err != nil {
		return "", fmt.Errorf("azure key vault: getting %q: %w", key, err)
	}

	if src.JSONKey != "" {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return "", fmt.Errorf("azure key vault: parsing %q as JSON: %w", key, err)
		}

		v, ok := m[src.JSONKey]
		if !ok {
			return "", fmt.Errorf("azure key vault: json_key %q not found in %q", src.JSONKey, key)
		}

		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("azure key vault: json_key %q in %q is not a string", src.JSONKey, key)
		}

		return str, nil
	}

	return value, nil
}
