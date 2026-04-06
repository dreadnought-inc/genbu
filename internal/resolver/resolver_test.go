package resolver

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/dreadnought-inc/genbu/internal/config"
	"github.com/dreadnought-inc/genbu/internal/provider"
)

// mockProvider is a test provider that returns predefined values.
type mockProvider struct {
	sourceType string
	values     map[string]string
}

func (m *mockProvider) Type() string { return m.sourceType }

func (m *mockProvider) Resolve(_ context.Context, src *config.SourceConfig) (string, error) {
	key := src.EffectiveKey()
	v, ok := m.values[key]
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

// mockPrefetchProvider implements both Provider and Prefetcher for testing.
type mockPrefetchProvider struct {
	mockProvider
	prefetchKeys []provider.PrefetchKey
	prefetchErr  error
}

func (m *mockPrefetchProvider) Prefetch(_ context.Context, keys []provider.PrefetchKey) error {
	m.prefetchKeys = append(m.prefetchKeys, keys...)
	return m.prefetchErr
}

func TestResolve_literalValues(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "APP_ENV", Value: "production"},
			{Name: "PORT", Value: "8080"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(result.Vars))
	}
	if result.Vars[0].Value != "production" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "production")
	}
	if result.Vars[1].Value != "8080" {
		t.Errorf("value = %q, want %q", result.Vars[1].Value, "8080")
	}
}

func TestResolve_envSource(t *testing.T) {
	t.Setenv("TEST_HOME", "/home/test")

	reg := provider.NewDefaultRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name:   "TEST_HOME",
				Source: &config.SourceConfig{Type: "env"},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "/home/test" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "/home/test")
	}
}

func TestResolve_noSourceNoValue(t *testing.T) {
	t.Setenv("EXISTING_VAR", "from-env")

	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "EXISTING_VAR"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "from-env" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "from-env")
	}
}

func TestResolve_withMockProvider(t *testing.T) {
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/app/db-host": "db.example.com",
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name: "DB_HOST",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/app/db-host",
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "db.example.com" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "db.example.com")
	}
}

func TestResolve_unknownProvider(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name: "SECRET",
				Source: &config.SourceConfig{
					Type:     "unknown-provider",
					SecretID: "my-secret",
				},
			},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestResolve_withGroups(t *testing.T) {
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/app/key1": "value1",
			"/app/key2": "value2",
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: &config.SourceConfig{
					Type:   "parameter",
					Region: "us-east-1",
				},
				Variables: []config.Variable{
					{
						Name:   "KEY1",
						Source: &config.SourceConfig{Key: "/app/key1"},
					},
					{
						Name:   "KEY2",
						Source: &config.SourceConfig{Key: "/app/key2"},
					},
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(result.Vars))
	}
	if result.Vars[0].Value != "value1" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "value1")
	}
	if result.Vars[1].Value != "value2" {
		t.Errorf("value = %q, want %q", result.Vars[1].Value, "value2")
	}
}

func TestResolve_defaultWhenEnvEmpty(t *testing.T) {
	reg := provider.NewDefaultRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name:    "UNSET_VAR",
				Default: "fallback_value",
				Source:  &config.SourceConfig{Type: "env"},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "fallback_value" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "fallback_value")
	}
}

func TestResolve_defaultNotUsedWhenEnvSet(t *testing.T) {
	t.Setenv("HAS_VALUE", "real_value")

	reg := provider.NewDefaultRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name:    "HAS_VALUE",
				Default: "fallback_value",
				Source:  &config.SourceConfig{Type: "env"},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "real_value" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "real_value")
	}
}

func TestResolve_defaultWithNoSource(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name:    "MISSING_VAR",
				Default: "default_val",
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "default_val" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "default_val")
	}
}

func TestResolve_variableReference(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "HOST", Value: "localhost"},
			{Name: "PORT", Value: "5432"},
			{Name: "DATABASE_URL", Value: "postgres://${HOST}:${PORT}/mydb"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[2].Value != "postgres://localhost:5432/mydb" {
		t.Errorf("value = %q, want %q", result.Vars[2].Value, "postgres://localhost:5432/mydb")
	}
}

func TestResolve_chainedReference(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: "hello"},
			{Name: "B", Value: "${A}-world"},
			{Name: "C", Value: "${B}-!"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[1].Value != "hello-world" {
		t.Errorf("B = %q, want %q", result.Vars[1].Value, "hello-world")
	}
	if result.Vars[2].Value != "hello-world-!" {
		t.Errorf("C = %q, want %q", result.Vars[2].Value, "hello-world-!")
	}
}

func TestResolve_circularReference(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: "${B}"},
			{Name: "B", Value: "${A}"},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for circular reference")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error = %q, want to contain 'circular reference'", err.Error())
	}
}

func TestResolve_selfReference(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: "${A}"},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for self reference")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error = %q, want to contain 'circular reference'", err.Error())
	}
}

func TestResolve_threeWayCycle(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: "${C}"},
			{Name: "B", Value: "${A}"},
			{Name: "C", Value: "${B}"},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for 3-way circular reference")
	}
}

func TestResolve_unknownRefKeptAsIs(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: "prefix-${UNKNOWN}-suffix"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "prefix-${UNKNOWN}-suffix" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "prefix-${UNKNOWN}-suffix")
	}
}

func TestResolve_referenceWithDefault(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "HOST", Value: "db.example.com"},
			{Name: "PORT", Default: "3306"},
			{Name: "DSN", Value: "${HOST}:${PORT}"},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[2].Value != "db.example.com:3306" {
		t.Errorf("value = %q, want %q", result.Vars[2].Value, "db.example.com:3306")
	}
}

func TestExtractAllRefs(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"no refs", "plain text", nil},
		{"single ref", "${FOO}", []string{"FOO"}},
		{"multiple refs", "${A}-${B}", []string{"A", "B"}},
		{"duplicate refs", "${A}-${A}", []string{"A", "A"}},
		{"embedded", "prefix-${VAR}-suffix", []string{"VAR"}},
		{"underscore", "${MY_VAR_1}", []string{"MY_VAR_1"}},
		{"invalid syntax not matched", "$FOO", nil},
		{"empty braces not matched", "${}", nil},
		{"expr var ref", `${{ sha256(SECRET) }}`, []string{"SECRET"}},
		{"mixed ref and expr", `${A}-${{ upper(B) }}`, []string{"A", "B"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAllRefs(tt.value)
			if len(got) != len(tt.want) {
				t.Fatalf("refs = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("refs[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolve_exprFunction(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "GREETING", Value: `${{ upper("hello") }}`},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "HELLO" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "HELLO")
	}
}

func TestResolve_exprWithVarRef(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "PASSWORD", Value: "s3cret"},
			{Name: "PASSWORD_HASH", Value: `${{ sha256(PASSWORD) }}`},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Vars[1].Value) != 64 {
		t.Errorf("sha256 output length = %d, want 64", len(result.Vars[1].Value))
	}
}

func TestResolve_exprNestedFunctions(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "ENCODED_HASH", Value: `${{ base64encode(sha256("hello")) }}`},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value == "" {
		t.Error("value should not be empty")
	}
}

func TestResolve_exprMixedWithVarRef(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "NAME", Value: "world"},
			{Name: "GREETING", Value: `hello-${{ upper(NAME) }}-${NAME}`},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[1].Value != "hello-WORLD-world" {
		t.Errorf("value = %q, want %q", result.Vars[1].Value, "hello-WORLD-world")
	}
}

func TestResolve_exprCircularViaExpr(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "A", Value: `${{ upper(B) }}`},
			{Name: "B", Value: `${{ lower(A) }}`},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for circular reference via expressions")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error = %q, want to contain 'circular reference'", err.Error())
	}
}

func TestResolve_preservesValidateConfig(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	boolTrue := true
	cfg := &config.Config{
		Version: "1",
		Defaults: &config.Defaults{
			Required: &boolTrue,
		},
		Variables: []config.Variable{
			{
				Name:  "APP_ENV",
				Value: "prod",
				Validate: &config.ValidateConfig{
					Enum: []string{"dev", "prod"},
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Defaults == nil || result.Defaults.Required == nil || !*result.Defaults.Required {
		t.Error("defaults should be preserved")
	}
	if result.Vars[0].Validate == nil {
		t.Error("validate config should be preserved")
	}
	if len(result.Vars[0].Validate.Enum) != 2 {
		t.Errorf("enum count = %d, want 2", len(result.Vars[0].Validate.Enum))
	}
}

func TestResolve_sourceKeyWithVarRef(t *testing.T) {
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/myapp/prod/db-host": "db.example.com",
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "APP_ENV", Value: "prod"},
			{
				Name: "DB_HOST",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/myapp/${APP_ENV}/db-host",
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[1].Value != "db.example.com" {
		t.Errorf("value = %q, want %q", result.Vars[1].Value, "db.example.com")
	}
}

func TestResolve_sourceKeyChainedRef(t *testing.T) {
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/myapp/production/us-east-1/db-host": "db.prod.example.com",
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "APP_ENV", Value: "production"},
			{Name: "REGION", Value: "us-east-1"},
			{
				Name: "DB_HOST",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/myapp/${APP_ENV}/${REGION}/db-host",
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[2].Value != "db.prod.example.com" {
		t.Errorf("value = %q, want %q", result.Vars[2].Value, "db.prod.example.com")
	}
}

func TestResolve_sourceKeyCircularRef(t *testing.T) {
	reg := provider.NewRegistry()
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name: "A",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/path/${B}",
				},
			},
			{
				Name: "B",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/path/${A}",
				},
			},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for circular reference in source keys")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error = %q, want to contain 'circular reference'", err.Error())
	}
}

func TestResolve_sourceKeyWithEnvDefault(t *testing.T) {
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/myapp/staging/db-host": "db.staging.example.com",
		},
	}

	reg := provider.NewDefaultRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{
				Name:    "APP_ENV",
				Source:  &config.SourceConfig{Type: "env"},
				Default: "staging",
			},
			{
				Name: "DB_HOST",
				Source: &config.SourceConfig{
					Type: "parameter",
					Key:  "/myapp/${APP_ENV}/db-host",
				},
			},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[1].Value != "db.staging.example.com" {
		t.Errorf("value = %q, want %q", result.Vars[1].Value, "db.staging.example.com")
	}
}

// --- Prefetch integration tests ---

func TestResolve_prefetchCalledForStaticKeys(t *testing.T) {
	mock := &mockPrefetchProvider{
		mockProvider: mockProvider{
			sourceType: "parameter",
			values: map[string]string{
				"/app/db-host": "db.example.com",
				"/app/db-port": "5432",
			},
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "DB_HOST", Source: &config.SourceConfig{Type: "parameter", Key: "/app/db-host"}},
			{Name: "DB_PORT", Source: &config.SourceConfig{Type: "parameter", Key: "/app/db-port"}},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Prefetch should have been called with both static keys
	if len(mock.prefetchKeys) != 2 {
		t.Fatalf("prefetchKeys count = %d, want 2", len(mock.prefetchKeys))
	}

	if result.Vars[0].Value != "db.example.com" {
		t.Errorf("DB_HOST = %q, want %q", result.Vars[0].Value, "db.example.com")
	}
}

func TestResolve_prefetchNotCalledForDynamicKeys(t *testing.T) {
	mock := &mockPrefetchProvider{
		mockProvider: mockProvider{
			sourceType: "parameter",
			values: map[string]string{
				"/app/prod/db-host": "db.example.com",
			},
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "ENV", Value: "prod"},
			{Name: "DB_HOST", Source: &config.SourceConfig{Type: "parameter", Key: "/app/${ENV}/db-host"}},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dynamic key should NOT be in prefetch
	if len(mock.prefetchKeys) != 0 {
		t.Errorf("prefetchKeys = %v, want empty (dynamic key should be excluded)", mock.prefetchKeys)
	}

	if result.Vars[1].Value != "db.example.com" {
		t.Errorf("DB_HOST = %q, want %q", result.Vars[1].Value, "db.example.com")
	}
}

func TestResolve_prefetchMixedStaticAndDynamic(t *testing.T) {
	mock := &mockPrefetchProvider{
		mockProvider: mockProvider{
			sourceType: "parameter",
			values: map[string]string{
				"/app/static-key":   "static-value",
				"/app/prod/dynamic": "dynamic-value",
			},
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "ENV", Value: "prod"},
			{Name: "STATIC", Source: &config.SourceConfig{Type: "parameter", Key: "/app/static-key"}},
			{Name: "DYNAMIC", Source: &config.SourceConfig{Type: "parameter", Key: "/app/${ENV}/dynamic"}},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only static key should be in prefetch
	if len(mock.prefetchKeys) != 1 {
		t.Fatalf("prefetchKeys count = %d, want 1", len(mock.prefetchKeys))
	}
	if mock.prefetchKeys[0].Key != "/app/static-key" {
		t.Errorf("prefetchKeys[0].Key = %q, want %q", mock.prefetchKeys[0].Key, "/app/static-key")
	}

	if result.Vars[1].Value != "static-value" {
		t.Errorf("STATIC = %q, want %q", result.Vars[1].Value, "static-value")
	}
	if result.Vars[2].Value != "dynamic-value" {
		t.Errorf("DYNAMIC = %q, want %q", result.Vars[2].Value, "dynamic-value")
	}
}

func TestResolve_prefetchProviderWithoutPrefetcher(t *testing.T) {
	// mockProvider does NOT implement Prefetcher
	mock := &mockProvider{
		sourceType: "parameter",
		values: map[string]string{
			"/app/key": "value",
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "VAR", Source: &config.SourceConfig{Type: "parameter", Key: "/app/key"}},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Vars[0].Value != "value" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "value")
	}
}

func TestResolve_prefetchWithRegion(t *testing.T) {
	mock := &mockPrefetchProvider{
		mockProvider: mockProvider{
			sourceType: "parameter",
			values: map[string]string{
				"/app/key": "value",
			},
		},
	}

	reg := provider.NewRegistry()
	reg.Register(mock)
	r := New(reg)

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "VAR", Source: &config.SourceConfig{Type: "parameter", Key: "/app/key", Region: "ap-northeast-1"}},
		},
	}

	_, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.prefetchKeys) != 1 {
		t.Fatalf("prefetchKeys count = %d, want 1", len(mock.prefetchKeys))
	}
	if mock.prefetchKeys[0].Region != "ap-northeast-1" {
		t.Errorf("region = %q, want %q", mock.prefetchKeys[0].Region, "ap-northeast-1")
	}
}

func TestResolve_prefetchEnvProviderSkipped(t *testing.T) {
	// Use a prefetch-capable mock for "parameter" to track calls,
	// plus the real env provider for env variables.
	mock := &mockPrefetchProvider{
		mockProvider: mockProvider{
			sourceType: "parameter",
			values:     map[string]string{},
		},
	}

	reg := provider.NewDefaultRegistry() // includes env provider
	reg.Register(mock)
	r := New(reg)

	t.Setenv("MY_VAR", "from-env")

	cfg := &config.Config{
		Version: "1",
		Variables: []config.Variable{
			{Name: "MY_VAR", Source: &config.SourceConfig{Type: "env"}},
		},
	}

	result, err := r.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// env type should not trigger prefetch on the parameter mock
	if len(mock.prefetchKeys) != 0 {
		t.Errorf("prefetchKeys = %v, want empty (env should be skipped)", mock.prefetchKeys)
	}
	if result.Vars[0].Value != "from-env" {
		t.Errorf("value = %q, want %q", result.Vars[0].Value, "from-env")
	}
}
