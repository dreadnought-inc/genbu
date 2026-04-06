package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/dreadnought-inc/genbu/internal/config"
)

type mockSMClient struct {
	secrets map[string]string
}

func (m *mockSMClient) GetSecretValue(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	id := *input.SecretId
	v, ok := m.secrets[id]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", id)
	}
	return &secretsmanager.GetSecretValueOutput{
		SecretString: &v,
	}, nil
}

func TestSecretProvider_Resolve_plainString(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string]string{
			"myapp/api-key": "supersecretkey",
		},
	}

	p := NewSecretProvider(client)

	got, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type: "secret",
		Key:  "myapp/api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "supersecretkey" {
		t.Errorf("value = %q, want %q", got, "supersecretkey")
	}
}

func TestSecretProvider_Resolve_jsonKey(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string]string{
			"myapp/creds": `{"username":"admin","password":"s3cret","api_key":"key123"}`,
		},
	}

	p := NewSecretProvider(client)

	tests := []struct {
		name    string
		jsonKey string
		want    string
		wantErr bool
	}{
		{name: "extract username", jsonKey: "username", want: "admin"},
		{name: "extract api_key", jsonKey: "api_key", want: "key123"},
		{name: "missing key", jsonKey: "nonexistent", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Resolve(context.Background(), &config.SourceConfig{
				Type:    "secret",
				Key:     "myapp/creds",
				JSONKey: tt.jsonKey,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecretProvider_Resolve_backwardCompat(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string]string{
			"myapp/api-key": "secret-value",
		},
	}

	p := NewSecretProvider(client)

	got, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type:     "secret",
		SecretID: "myapp/api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secret-value" {
		t.Errorf("value = %q, want %q", got, "secret-value")
	}
}

func TestSecretProvider_Resolve_missingKey(t *testing.T) {
	p := NewSecretProvider(&mockSMClient{})

	_, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type: "secret",
	})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestSecretProvider_Resolve_secretNotFound(t *testing.T) {
	p := NewSecretProvider(&mockSMClient{secrets: map[string]string{}})

	_, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type: "secret",
		Key:  "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent secret")
	}
}

func TestSecretProvider_Resolve_invalidJSON(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string]string{"myapp/bad": "not-json"},
	}
	p := NewSecretProvider(client)

	_, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type:    "secret",
		Key:     "myapp/bad",
		JSONKey: "key",
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSecretProvider_Type(t *testing.T) {
	p := &SecretProvider{}
	if p.Type() != "secret" {
		t.Errorf("Type() = %q, want %q", p.Type(), "secret")
	}
}
