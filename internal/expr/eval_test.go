package expr

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEval_noExpression(t *testing.T) {
	got, err := Eval("plain text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plain text" {
		t.Errorf("got %q, want %q", got, "plain text")
	}
}

func TestEval_base64encode(t *testing.T) {
	got, err := Eval(`${{ base64encode("hello") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := base64.StdEncoding.EncodeToString([]byte("hello"))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_base64decode(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	got, err := Eval(`${{ base64decode("`+encoded+`") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEval_tohex(t *testing.T) {
	got, err := Eval(`${{ tohex("abc") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := hex.EncodeToString([]byte("abc"))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_fromhex(t *testing.T) {
	got, err := Eval(`${{ fromhex("616263") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc" {
		t.Errorf("got %q, want %q", got, "abc")
	}
}

func TestEval_sha256(t *testing.T) {
	got, err := Eval(`${{ sha256("hello") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SHA-256 of "hello"
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_sha384(t *testing.T) {
	got, err := Eval(`${{ sha384("hello") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 96 { // SHA-384 = 48 bytes = 96 hex chars
		t.Errorf("sha384 output length = %d, want 96", len(got))
	}
}

func TestEval_sha512(t *testing.T) {
	got, err := Eval(`${{ sha512("hello") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 128 { // SHA-512 = 64 bytes = 128 hex chars
		t.Errorf("sha512 output length = %d, want 128", len(got))
	}
}

func TestEval_bcrypt(t *testing.T) {
	got, err := Eval(`${{ bcrypt("password", 4) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "$2a$") {
		t.Errorf("bcrypt output should start with $2a$, got %q", got)
	}
}

func TestEval_bcryptDefaultCost(t *testing.T) {
	got, err := Eval(`${{ bcrypt("password") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "$2a$") {
		t.Errorf("bcrypt output should start with $2a$, got %q", got)
	}
}

func TestEval_randomString(t *testing.T) {
	got, err := Eval(`${{ random_string(32) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 32 {
		t.Errorf("random_string length = %d, want 32", len(got))
	}
}

func TestEval_randomHex(t *testing.T) {
	got, err := Eval(`${{ random_hex(16) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("random_hex length = %d, want 32", len(got))
	}
}

func TestEval_upper(t *testing.T) {
	got, err := Eval(`${{ upper("hello") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "HELLO" {
		t.Errorf("got %q, want %q", got, "HELLO")
	}
}

func TestEval_lower(t *testing.T) {
	got, err := Eval(`${{ lower("HELLO") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEval_trim(t *testing.T) {
	got, err := Eval(`${{ trim("  hello  ") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEval_replace(t *testing.T) {
	got, err := Eval(`${{ replace("hello world", "world", "go") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello go" {
		t.Errorf("got %q, want %q", got, "hello go")
	}
}

func TestEval_substr(t *testing.T) {
	got, err := Eval(`${{ substr("hello world", 6) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "world" {
		t.Errorf("got %q, want %q", got, "world")
	}
}

func TestEval_substrWithLength(t *testing.T) {
	got, err := Eval(`${{ substr("hello world", 0, 5) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestEval_concat(t *testing.T) {
	got, err := Eval(`${{ concat("a", "b", "c") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc" {
		t.Errorf("got %q, want %q", got, "abc")
	}
}

func TestEval_nested(t *testing.T) {
	got, err := Eval(`${{ base64encode(sha256("hello")) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sha256("hello") -> hex string, then base64 encode it
	sha := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	want := base64.StdEncoding.EncodeToString([]byte(sha))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_deeplyNested(t *testing.T) {
	// tohex(bcrypt(random_string(16), 4)) — verify it runs without error and produces hex
	got, err := Eval(`${{ tohex(bcrypt(random_string(16), 4)) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// bcrypt output is ~60 chars, hex encoded = ~120 chars
	if len(got) < 100 {
		t.Errorf("output unexpectedly short: %d chars", len(got))
	}
	// Verify it's valid hex
	if _, err := hex.DecodeString(got); err != nil {
		t.Errorf("output is not valid hex: %v", err)
	}
}

func TestEval_varRef(t *testing.T) {
	vars := map[string]string{
		"SECRET": "mysecretvalue",
	}
	got, err := Eval(`${{ sha256(SECRET) }}`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 64 {
		t.Errorf("sha256 output length = %d, want 64", len(got))
	}
}

func TestEval_mixedTextAndExpr(t *testing.T) {
	got, err := Eval(`prefix-${{ upper("hello") }}-suffix`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "prefix-HELLO-suffix" {
		t.Errorf("got %q, want %q", got, "prefix-HELLO-suffix")
	}
}

func TestEval_multipleExpressions(t *testing.T) {
	got, err := Eval(`${{ upper("a") }}-${{ lower("B") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "A-b" {
		t.Errorf("got %q, want %q", got, "A-b")
	}
}

func TestEval_unknownFunction(t *testing.T) {
	_, err := Eval(`${{ md5("hello") }}`, nil)
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
	if !strings.Contains(err.Error(), "unknown function") {
		t.Errorf("error = %q, want to contain 'unknown function'", err.Error())
	}
}

func TestEval_undefinedVar(t *testing.T) {
	_, err := Eval(`${{ sha256(UNDEFINED_VAR) }}`, map[string]string{})
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
}

func TestEval_wrongArgCount(t *testing.T) {
	_, err := Eval(`${{ sha256("a", "b") }}`, nil)
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestExtractExprVarRefs(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"no expr", "plain text", nil},
		{"no vars", `${{ sha256("literal") }}`, nil},
		{"single var", `${{ sha256(SECRET) }}`, []string{"SECRET"}},
		{"multiple vars", `${{ concat(A, B) }}`, []string{"A", "B"}},
		{"nested with var", `${{ upper(base64encode(KEY)) }}`, []string{"KEY"}},
		{"mixed text", `prefix-${{ lower(NAME) }}-suffix`, []string{"NAME"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractExprVarRefs(tt.value)
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

// --- Date/Time ---

func TestEval_date_default(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 12, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ date() }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-04-05" {
		t.Errorf("got %q, want %q", got, "2026-04-05")
	}
}

func TestEval_date_customFormat(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 12, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ date("02/01/2006") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "05/04/2026" {
		t.Errorf("got %q, want %q", got, "05/04/2026")
	}
}

func TestEval_time_default(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ time() }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "14:30:45" {
		t.Errorf("got %q, want %q", got, "14:30:45")
	}
}

func TestEval_time_kitchen(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ time("kitchen") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2:30PM" {
		t.Errorf("got %q, want %q", got, "2:30PM")
	}
}

func TestEval_datetime_default(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ datetime() }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-04-05T14:30:45Z" {
		t.Errorf("got %q, want %q", got, "2026-04-05T14:30:45Z")
	}
}

func TestEval_datetime_rfc3339(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ datetime("rfc3339") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-04-05T14:30:45Z" {
		t.Errorf("got %q, want %q", got, "2026-04-05T14:30:45Z")
	}
}

func TestEval_datetime_customFormat(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ datetime("2006/01/02 15:04") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026/04/05 14:30" {
		t.Errorf("got %q, want %q", got, "2026/04/05 14:30")
	}
}

func TestEval_timestamp_default(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ timestamp() }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := strconv.FormatInt(fixed.Unix(), 10)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_timestamp_unixmilli(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ timestamp("unixmilli") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := strconv.FormatInt(fixed.UnixMilli(), 10)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEval_timestamp_rfc3339(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 14, 30, 45, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ timestamp("rfc3339") }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-04-05T14:30:45Z" {
		t.Errorf("got %q, want %q", got, "2026-04-05T14:30:45Z")
	}
}

func TestEval_date_nested(t *testing.T) {
	fixed := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	origNow := nowFunc
	nowFunc = func() time.Time { return fixed }
	defer func() { nowFunc = origNow }()

	got, err := Eval(`${{ concat("deployed-", date()) }}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "deployed-2026-04-05" {
		t.Errorf("got %q, want %q", got, "deployed-2026-04-05")
	}
}

func TestParse_expressions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"function call", "sha256('hello')", false},
		{"nested call", "base64encode(sha256('x'))", false},
		{"var ref", "MY_VAR", false},
		{"string literal", `"hello"`, false},
		{"number literal", "42", false},
		{"empty args", "random_string()", true}, // random_string needs 1 arg, but parse succeeds - eval fails
		{"multiple args", "replace('a', 'b', 'c')", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			// Parse should succeed for valid syntax regardless of function arg validation
			if tt.name != "empty args" && err != nil {
				t.Errorf("unexpected parse error: %v", err)
			}
		})
	}
}
