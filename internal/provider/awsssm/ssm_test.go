package awsssm

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

func TestProvider_Resolve(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/db-host": "db.example.com",
			"/app/db-port": "5432",
		},
	}

	p := New(client)

	tests := []struct {
		name    string
		src     *config.SourceConfig
		want    string
		wantErr bool
	}{
		{
			name: "existing parameter",
			src:  &config.SourceConfig{Type: "aws-ssm", Path: "/app/db-host"},
			want: "db.example.com",
		},
		{
			name: "another parameter",
			src:  &config.SourceConfig{Type: "aws-ssm", Path: "/app/db-port"},
			want: "5432",
		},
		{
			name:    "missing parameter",
			src:     &config.SourceConfig{Type: "aws-ssm", Path: "/app/missing"},
			wantErr: true,
		},
		{
			name:    "empty path",
			src:     &config.SourceConfig{Type: "aws-ssm"},
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

func TestProvider_Type(t *testing.T) {
	p := &Provider{}
	if p.Type() != "aws-ssm" {
		t.Errorf("Type() = %q, want %q", p.Type(), "aws-ssm")
	}
}
