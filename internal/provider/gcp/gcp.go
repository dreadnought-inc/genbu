package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/dreadnought-inc/genbu/internal/config"
)

// SMClient is the subset of the GCP Secret Manager API used by this provider.
type SMClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

// Config holds initialized GCP SDK clients.
type Config struct {
	SM SMClient
}

// NewDefaultConfig creates GCP clients using Application Default Credentials.
func NewDefaultConfig(ctx context.Context) (*Config, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating gcp secret manager client: %w", err)
	}
	return &Config{SM: &smClientWrapper{client: client}}, nil
}

// smClientWrapper adapts the real GCP client to the SMClient interface.
type smClientWrapper struct {
	client *secretmanager.Client
}

func (w *smClientWrapper) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return w.client.AccessSecretVersion(ctx, req)
}

// ParameterProvider fetches values from GCP Secret Manager.
// In GCP, both "parameter" and "secret" source types use Secret Manager.
type ParameterProvider struct {
	client SMClient
}

// NewParameterProvider creates a ParameterProvider.
func NewParameterProvider(client SMClient) *ParameterProvider {
	return &ParameterProvider{client: client}
}

// Type returns "parameter".
func (p *ParameterProvider) Type() string {
	return "parameter"
}

// Resolve fetches a secret version from GCP Secret Manager.
// key format: projects/{project}/secrets/{secret}/versions/{version}
// If no version suffix, "/versions/latest" is appended automatically.
func (p *ParameterProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	return resolveSecret(ctx, p.client, src)
}

// SecretProvider fetches values from GCP Secret Manager.
type SecretProvider struct {
	client SMClient
}

// NewSecretProvider creates a SecretProvider.
func NewSecretProvider(client SMClient) *SecretProvider {
	return &SecretProvider{client: client}
}

// Type returns "secret".
func (p *SecretProvider) Type() string {
	return "secret"
}

// Resolve fetches a secret version from GCP Secret Manager.
func (p *SecretProvider) Resolve(ctx context.Context, src *config.SourceConfig) (string, error) {
	return resolveSecret(ctx, p.client, src)
}

func resolveSecret(ctx context.Context, client SMClient, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	if key == "" {
		return "", fmt.Errorf("gcp secret manager: key is required")
	}

	name := normalizeSecretName(key)

	resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("gcp secret manager: accessing %q: %w", key, err)
	}

	if resp.Payload == nil {
		return "", fmt.Errorf("gcp secret manager: %q has no payload", key)
	}

	value := string(resp.Payload.Data)

	if src.JSONKey != "" {
		var m map[string]interface{}
		if err := json.Unmarshal(resp.Payload.Data, &m); err != nil {
			return "", fmt.Errorf("gcp secret manager: parsing %q as JSON: %w", key, err)
		}

		v, ok := m[src.JSONKey]
		if !ok {
			return "", fmt.Errorf("gcp secret manager: json_key %q not found in %q", src.JSONKey, key)
		}

		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("gcp secret manager: json_key %q in %q is not a string", src.JSONKey, key)
		}

		return str, nil
	}

	return value, nil
}

// normalizeSecretName ensures the key has a /versions/ suffix.
func normalizeSecretName(key string) string {
	if strings.Contains(key, "/versions/") {
		return key
	}
	return key + "/versions/latest"
}
