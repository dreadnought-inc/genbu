package gcp

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/dreadnought-inc/genbu/internal/config"
)

type mockSMClient struct {
	secrets map[string][]byte
}

func (m *mockSMClient) AccessSecretVersion(_ context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	data, ok := m.secrets[req.Name]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", req.Name)
	}
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{Data: data},
	}, nil
}

func TestParameterProvider_Resolve(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string][]byte{
			"projects/myproject/secrets/db-host/versions/latest": []byte("db.example.com"),
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
			name: "full path",
			src:  &config.SourceConfig{Type: "parameter", Key: "projects/myproject/secrets/db-host/versions/latest"},
			want: "db.example.com",
		},
		{
			name: "auto append versions/latest",
			src:  &config.SourceConfig{Type: "parameter", Key: "projects/myproject/secrets/db-host"},
			want: "db.example.com",
		},
		{
			name:    "missing secret",
			src:     &config.SourceConfig{Type: "parameter", Key: "projects/myproject/secrets/missing"},
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

func TestSecretProvider_Resolve_jsonKey(t *testing.T) {
	client := &mockSMClient{
		secrets: map[string][]byte{
			"projects/p/secrets/creds/versions/latest": []byte(`{"user":"admin","pass":"s3cret"}`),
		},
	}

	p := NewSecretProvider(client)

	got, err := p.Resolve(context.Background(), &config.SourceConfig{
		Type:    "secret",
		Key:     "projects/p/secrets/creds",
		JSONKey: "pass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "s3cret" {
		t.Errorf("value = %q, want %q", got, "s3cret")
	}
}

func TestSecretProvider_Type(t *testing.T) {
	p := &SecretProvider{}
	if p.Type() != "secret" {
		t.Errorf("Type() = %q, want %q", p.Type(), "secret")
	}
}

func TestParameterProvider_Type(t *testing.T) {
	p := &ParameterProvider{}
	if p.Type() != "parameter" {
		t.Errorf("Type() = %q, want %q", p.Type(), "parameter")
	}
}

func TestNormalizeSecretName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"projects/p/secrets/s", "projects/p/secrets/s/versions/latest"},
		{"projects/p/secrets/s/versions/1", "projects/p/secrets/s/versions/1"},
		{"projects/p/secrets/s/versions/latest", "projects/p/secrets/s/versions/latest"},
	}

	for _, tt := range tests {
		got := normalizeSecretName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSecretName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
