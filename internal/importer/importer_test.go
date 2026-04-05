package importer

import (
	"strings"
	"testing"
)

// --- Dotenv ---

func TestDotenv_basic(t *testing.T) {
	input := `
DATABASE_URL=postgres://localhost:5432/mydb
API_KEY=abc123
PORT=8080
`
	imp := &DotenvImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("vars count = %d, want 3", len(vars))
	}
	if vars[0].Name != "DATABASE_URL" || vars[0].Value != "postgres://localhost:5432/mydb" {
		t.Errorf("vars[0] = %+v", vars[0])
	}
}

func TestDotenv_commentsAndEmpty(t *testing.T) {
	input := `
# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`
	imp := &DotenvImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(vars))
	}
}

func TestDotenv_quotedValues(t *testing.T) {
	input := `
DOUBLE="hello world"
SINGLE='single quoted'
NONE=no quotes
`
	imp := &DotenvImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars[0].Value != "hello world" {
		t.Errorf("double quoted = %q, want %q", vars[0].Value, "hello world")
	}
	if vars[1].Value != "single quoted" {
		t.Errorf("single quoted = %q, want %q", vars[1].Value, "single quoted")
	}
	if vars[2].Value != "no quotes" {
		t.Errorf("unquoted = %q, want %q", vars[2].Value, "no quotes")
	}
}

func TestDotenv_exportPrefix(t *testing.T) {
	input := `export MY_VAR=hello`
	imp := &DotenvImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 1 || vars[0].Name != "MY_VAR" {
		t.Errorf("vars = %+v", vars)
	}
}

// --- INI ---

func TestINI_basic(t *testing.T) {
	input := `
[database]
host = localhost
port = 5432

[api]
key = abc123
`
	imp := &INIImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("vars count = %d, want 3", len(vars))
	}

	expected := map[string]string{
		"DATABASE_HOST": "localhost",
		"DATABASE_PORT": "5432",
		"API_KEY":       "abc123",
	}

	for _, v := range vars {
		if want, ok := expected[v.Name]; ok {
			if v.Value != want {
				t.Errorf("%s = %q, want %q", v.Name, v.Value, want)
			}
		} else {
			t.Errorf("unexpected var: %s", v.Name)
		}
	}
}

func TestINI_noSection(t *testing.T) {
	input := `
key1 = value1
key2 = value2
`
	imp := &INIImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(vars))
	}
	if vars[0].Name != "KEY1" {
		t.Errorf("name = %q, want %q", vars[0].Name, "KEY1")
	}
}

func TestINI_comments(t *testing.T) {
	input := `
; ini comment
# hash comment
[section]
key = value
`
	imp := &INIImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 1 {
		t.Fatalf("vars count = %d, want 1", len(vars))
	}
}

// --- TOML ---

func TestTOML_basic(t *testing.T) {
	input := `
[database]
host = "localhost"
port = 5432

[api]
key = "abc123"
`
	imp := &TOMLImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"DATABASE_HOST": "localhost",
		"DATABASE_PORT": "5432",
		"API_KEY":       "abc123",
	}

	for _, v := range vars {
		if want, ok := expected[v.Name]; ok {
			if v.Value != want {
				t.Errorf("%s = %q, want %q", v.Name, v.Value, want)
			}
		}
	}
}

func TestTOML_nested(t *testing.T) {
	input := `
[database.credentials]
user = "admin"
password = "secret"
`
	imp := &TOMLImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"DATABASE_CREDENTIALS_PASSWORD": "secret",
		"DATABASE_CREDENTIALS_USER":     "admin",
	}

	if len(vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(vars))
	}

	for _, v := range vars {
		if want, ok := expected[v.Name]; ok {
			if v.Value != want {
				t.Errorf("%s = %q, want %q", v.Name, v.Value, want)
			}
		} else {
			t.Errorf("unexpected var: %s=%s", v.Name, v.Value)
		}
	}
}

func TestTOML_topLevel(t *testing.T) {
	input := `
app_name = "myapp"
debug = true
`
	imp := &TOMLImporter{}
	vars, err := imp.Import(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vars) != 2 {
		t.Fatalf("vars count = %d, want 2", len(vars))
	}
}

// --- GenerateYAML ---

func TestGenerateYAML(t *testing.T) {
	vars := []EnvVar{
		{Name: "APP_ENV", Value: "production"},
		{Name: "PORT", Value: "8080"},
	}

	out, err := GenerateYAML(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yaml := string(out)
	if !strings.Contains(yaml, "version:") {
		t.Error("output should contain version")
	}
	if !strings.Contains(yaml, "APP_ENV") {
		t.Error("output should contain APP_ENV")
	}
	if !strings.Contains(yaml, "PORT") {
		t.Error("output should contain PORT")
	}
}

// --- FormatFromExtension ---

func TestFormatFromExtension(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"config.toml", "toml"},
		{"config.ini", "ini"},
		{"config.cfg", "ini"},
		{".env", "dotenv"},
		{"app.env", "dotenv"},
		{"config.yaml", ""},
		{"noext", ""},
	}

	for _, tt := range tests {
		got := FormatFromExtension(tt.path)
		if got != tt.want {
			t.Errorf("FormatFromExtension(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
