package azure

import (
	"context"
	"fmt"
	"testing"

	"github.com/dreadnought-inc/genbu/internal/config"
)

type mockAppConfigClient struct {
	settings map[string]string
}

func (m *mockAppConfigClient) GetSetting(_ context.Context, key string) (string, error) {
	v, ok := m.settings[key]
	if !ok {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	return v, nil
}

type mockKeyVaultClient struct {
	secrets map[string]string
}

func (m *mockKeyVaultClient) GetSecret(_ context.Context, name string) (string, error) {
	v, ok := m.secrets[name]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", name)
	}
	return v, nil
}

func TestParameterProvider_Resolve(t *testing.T) {
	client := &mockAppConfigClient{
		settings: map[string]string{
			"myapp/db-host": "db.example.com",
			"myapp/db-port": "5432",
		},
	}

	p := NewParameterProvider(client)

	tests := []struct {
		name    string
		src     *config.SourceConfig
		want    string
		wantErr bool
	}{
		{
			name: "existing setting",
			src:  &config.SourceConfig{Type: "parameter", Key: "myapp/db-host"},
			want: "db.example.com",
		},
		{
			name:    "missing setting",
			src:     &config.SourceConfig{Type: "parameter", Key: "myapp/missing"},
			wantErr: true,
		},
		{
			name:    "empty key",
			src:     &config.SourceConfig{Type: "parameter"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Resolve(context.Background(), tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecretProvider_Resolve_plainString(t *testing.T) {
	client := &mockKeyVaultClient{
		secrets: map[string]string{
			"api-key": "supersecretkey",
		},
	}

	p := NewSecretProvider(client)

	got, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type: "secret",
		Key:  "api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "supersecretkey" {
		t.Errorf("value = %q, want %q", got, "supersecretkey")
	}
}

func TestSecretProvider_Resolve_jsonKey(t *testing.T) {
	client := &mockKeyVaultClient{
		secrets: map[string]string{
			"creds": `{"user":"admin","pass":"s3cret"}`,
		},
	}

	p := NewSecretProvider(client)

	got, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type:    "secret",
		Key:     "creds",
		JSONKey: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "s3cret" {
		t.Errorf("value = %q, want %q", got, "s3cret")
	}
}

func TestSecretProvider_Resolve_missingKey(t *testing.T) {
	p := NewSecretProvider(&mockKeyVaultClient{secrets: map[string]string{}})

	_, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "secret"})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestParameterProvider_Type(t *testing.T) {
	p := &ParameterProvider{}
	if p.Type() != "parameter" {
		t.Errorf("Type() = %q, want %q", p.Type(), "parameter")
	}
}

func TestSecretProvider_Type(t *testing.T) {
	p := &SecretProvider{}
	if p.Type() != "secret" {
		t.Errorf("Type() = %q, want %q", p.Type(), "secret")
	}
}
