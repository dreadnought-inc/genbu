package aws

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
)

type mockSSMClient struct {
	params          map[string]string
	getCalls        int
	getByPathCalls  int
	getByPathPageFn func(path string, page int) ([]types.Parameter, *string)
}

func (m *mockSSMClient) GetParameter(_ context.Context, input *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	m.getCalls++
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

func (m *mockSSMClient) GetParametersByPath(_ context.Context, input *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	m.getByPathCalls++
	path := *input.Path

	// Use custom pagination function if set
	if m.getByPathPageFn != nil {
		page := 0
		if input.NextToken != nil {
			// Parse page number from token
			_, _ = fmt.Sscanf(*input.NextToken, "page%d", &page)
		}
		params, nextToken := m.getByPathPageFn(path, page)
		return &ssm.GetParametersByPathOutput{
			Parameters: params,
			NextToken:  nextToken,
		}, nil
	}

	// Default: return all params matching the path prefix
	var params []types.Parameter
	for name, value := range m.params {
		if strings.HasPrefix(name, path) {
			n := name
			v := value
			params = append(params, types.Parameter{Name: &n, Value: &v})
		}
	}
	return &ssm.GetParametersByPathOutput{
		Parameters: params,
	}, nil
}

// --- Prefetch Tests ---

func TestParameterProvider_Prefetch_Basic(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/prod/db-host": "db.example.com",
			"/app/prod/db-port": "5432",
			"/app/prod/db-user": "admin",
		},
	}

	p := NewParameterProvider(client)

	err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/db-host"},
		{Key: "/app/prod/db-port"},
		{Key: "/app/prod/db-user"},
	})
	if err != nil {
		t.Fatalf("Prefetch error: %v", err)
	}

	if client.getByPathCalls != 1 {
		t.Errorf("getByPathCalls = %d, want 1 (single batch for /app/prod/)", client.getByPathCalls)
	}

	// Resolve should use cache, not call GetParameter
	for _, key := range []string{"/app/prod/db-host", "/app/prod/db-port", "/app/prod/db-user"} {
		_, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: key})
		if err != nil {
			t.Errorf("Resolve(%s) error: %v", key, err)
		}
	}
	if client.getCalls != 0 {
		t.Errorf("getCalls = %d, want 0 (all served from cache)", client.getCalls)
	}
}

func TestParameterProvider_Prefetch_CacheHit(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/prod/secret": "cached-value",
		},
	}

	p := NewParameterProvider(client)
	if err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/secret"},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: "/app/prod/secret"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "cached-value" {
		t.Errorf("value = %q, want %q", got, "cached-value")
	}
	if client.getCalls != 0 {
		t.Errorf("getCalls = %d, want 0", client.getCalls)
	}
}

func TestParameterProvider_Prefetch_CacheMiss(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/prod/db-host": "db.example.com",
			"/other/key":        "other-value",
		},
	}

	p := NewParameterProvider(client)
	if err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/db-host"},
	}); err != nil {
		t.Fatal(err)
	}

	// This key wasn't prefetched → falls back to GetParameter
	got, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: "/other/key"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "other-value" {
		t.Errorf("value = %q, want %q", got, "other-value")
	}
	if client.getCalls != 1 {
		t.Errorf("getCalls = %d, want 1", client.getCalls)
	}
}

func TestParameterProvider_Prefetch_Pagination(t *testing.T) {
	client := &mockSSMClient{
		getByPathPageFn: func(path string, page int) ([]types.Parameter, *string) {
			if path != "/app/prod/" {
				return nil, nil
			}
			switch page {
			case 0:
				n1, v1 := "/app/prod/key1", "val1"
				n2, v2 := "/app/prod/key2", "val2"
				next := "page1"
				return []types.Parameter{
					{Name: &n1, Value: &v1},
					{Name: &n2, Value: &v2},
				}, &next
			case 1:
				n3, v3 := "/app/prod/key3", "val3"
				return []types.Parameter{
					{Name: &n3, Value: &v3},
				}, nil
			default:
				return nil, nil
			}
		},
	}

	p := NewParameterProvider(client)
	err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/key1"},
	})
	if err != nil {
		t.Fatalf("Prefetch error: %v", err)
	}

	if client.getByPathCalls != 2 {
		t.Errorf("getByPathCalls = %d, want 2 (2 pages)", client.getByPathCalls)
	}

	// All 3 keys from both pages should be cached
	for _, tc := range []struct{ key, want string }{
		{"/app/prod/key1", "val1"},
		{"/app/prod/key2", "val2"},
		{"/app/prod/key3", "val3"},
	} {
		got, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: tc.key})
		if err != nil {
			t.Errorf("Resolve(%s) error: %v", tc.key, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Resolve(%s) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestParameterProvider_Prefetch_WithRegion(t *testing.T) {
	usParams := map[string]string{"/app/db-host": "db-us"}
	euParams := map[string]string{"/app/db-host": "db-eu"}

	client := &mockSSMClient{
		getByPathPageFn: func(_ string, _ int) ([]types.Parameter, *string) {
			return nil, nil
		},
	}

	// Use a more targeted test: verify that cache keys are region-aware
	p := NewParameterProvider(client)
	p.cache = make(map[string]string)
	// Manually populate cache as if Prefetch ran for different regions
	for name, value := range usParams {
		p.cache[cacheKey("us-east-1", name)] = value
	}
	for name, value := range euParams {
		p.cache[cacheKey("eu-west-1", name)] = value
	}

	gotUS, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: "/app/db-host", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotUS != "db-us" {
		t.Errorf("us value = %q, want %q", gotUS, "db-us")
	}

	gotEU, err := p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: "/app/db-host", Region: "eu-west-1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotEU != "db-eu" {
		t.Errorf("eu value = %q, want %q", gotEU, "db-eu")
	}
}

func TestParameterProvider_Prefetch_DifferentPathGroups(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/app/prod/db-host": "db.example.com",
			"/cache/prod/url":   "redis://localhost",
		},
	}

	p := NewParameterProvider(client)
	err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/db-host"},
		{Key: "/cache/prod/url"},
	})
	if err != nil {
		t.Fatalf("Prefetch error: %v", err)
	}

	// Two different parent paths → two GetParametersByPath calls
	if client.getByPathCalls != 2 {
		t.Errorf("getByPathCalls = %d, want 2", client.getByPathCalls)
	}
}

func TestParameterProvider_Prefetch_EmptyResult(t *testing.T) {
	client := &mockSSMClient{
		params: map[string]string{
			"/other/key": "value",
		},
	}

	p := NewParameterProvider(client)
	// Prefetch a path that has no matching parameters
	err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/empty/path/key"},
	})
	if err != nil {
		t.Fatalf("Prefetch error: %v", err)
	}

	// Resolve should fall back to GetParameter
	_, err = p.Resolve(context.Background(), &config.SourceConfig{Type: "parameter", Key: "/other/key"})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if client.getCalls != 1 {
		t.Errorf("getCalls = %d, want 1", client.getCalls)
	}
}

func TestParameterProvider_Prefetch_Error(t *testing.T) {
	errClient := &errorSSMClient{err: fmt.Errorf("access denied")}
	p := NewParameterProvider(errClient)

	err := p.Prefetch(context.Background(), []provider.PrefetchKey{
		{Key: "/app/prod/key"},
	})
	if err == nil {
		t.Fatal("expected error from Prefetch")
	}
	if !strings.Contains(err.Error(), "access denied") {
		t.Errorf("error = %q, want to contain 'access denied'", err.Error())
	}
}

type errorSSMClient struct {
	err error
}

func (m *errorSSMClient) GetParameter(_ context.Context, _ *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return nil, m.err
}

func (m *errorSSMClient) GetParametersByPath(_ context.Context, _ *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	return nil, m.err
}

// --- parentPath Tests ---

func TestParentPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/a/b/c", "/a/b/"},
		{"/a/b", "/a/"},
		{"/a", "/"},
		{"a", ""},
		{"", ""},
		{"/app/prod/db-host", "/app/prod/"},
		{"/single/", "/single/"},
	}

	for _, tt := range tests {
		got := parentPath(tt.input)
		if got != tt.want {
			t.Errorf("parentPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Original Resolve Tests (backward compat) ---

func TestParameterProvider_Resolve_NoPrefetch(t *testing.T) {
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
