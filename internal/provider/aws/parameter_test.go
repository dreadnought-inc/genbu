package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/dreadnought-inc/genbu/internal/config"
)

type mockSSMClient struct {
	params map[string]string
}

func (m *mockSSMClient) GetParameter(_ context.Context, input *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	name := *input.Name
	v, ok := m.params[name]
	if !ok {
		return nil, fmt.Errorf("parameter not found: %s", name)
	}
	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Name:  &name,
			Value: &v,
		},
	}, nil
}

func TestParameterProvider_Resolve(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/db-host": "db.example.com",
			"/app/db-port": "5432",
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
			name: "existing parameter",
			src:  &config.SourceConfig{Type: "parameter", Key: "/app/db-host"},
			want: "db.example.com",
		},
		{
			name: "another parameter",
			src:  &config.SourceConfig{Type: "parameter", Key: "/app/db-port"},
			want: "5432",
		},
		{
			name:    "missing parameter",
			src:     &config.SourceConfig{Type: "parameter", Key: "/app/missing"},
			wantErr: true,
		},
		{
			name:    "empty key",
			src:     &config.SourceConfig{Type: "parameter"},
			wantErr: true,
		},
		{
			name: "backward compat path fallback",
			src:  &config.SourceConfig{Type: "parameter", Path: "/app/db-host"},
			want: "db.example.com",
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

func TestParameterProvider_Type(t *testing.T) {
	p := &ParameterProvider{}
	if p.Type() != "parameter" {
		t.Errorf("Type() = %q, want %q", p.Type(), "parameter")
	}
}
