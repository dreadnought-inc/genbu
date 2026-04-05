package expr

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Func is a built-in function implementation.
type Func struct {
	MinArgs int
	MaxArgs int // -1 for unlimited
	Call    func(args []string) (string, error)
}

// Registry returns the built-in function registry.
func Registry() map[string]Func {
	return map[string]Func{
		// Encoding
		"base64encode": {1, 1, fnBase64Encode},
		"base64decode": {1, 1, fnBase64Decode},
		"tohex":        {1, 1, fnToHex},
		"fromhex":      {1, 1, fnFromHex},

		// Hashing (secure algorithms only — MD5 and SHA-1 are intentionally excluded)
		"sha256": {1, 1, fnSHA256},
		"sha384": {1, 1, fnSHA384},
		"sha512": {1, 1, fnSHA512},

		// Password hashing
		"bcrypt": {1, 2, fnBcrypt},

		// Random
		"random_string": {1, 1, fnRandomString},
		"random_hex":    {1, 1, fnRandomHex},

		// Date/Time
		"date":      {0, 1, fnDate},
		"time":      {0, 1, fnTime},
		"datetime":  {0, 1, fnDatetime},
		"timestamp": {0, 1, fnTimestamp},

		// String
		"upper":   {1, 1, fnUpper},
		"lower":   {1, 1, fnLower},
		"trim":    {1, 1, fnTrim},
		"replace": {3, 3, fnReplace},
		"substr":  {2, 3, fnSubstr},
		"concat":  {1, -1, fnConcat},
	}
}

// --- Encoding ---

func fnBase64Encode(args []string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(args[0])), nil
}

func fnBase64Decode(args []string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(args[0])
	if err != nil {
		return "", fmt.Errorf("base64decode: %w", err)
	}
	return string(b), nil
}

func fnToHex(args []string) (string, error) {
	return hex.EncodeToString([]byte(args[0])), nil
}

func fnFromHex(args []string) (string, error) {
	b, err := hex.DecodeString(args[0])
	if err != nil {
		return "", fmt.Errorf("fromhex: %w", err)
	}
	return string(b), nil
}

// --- Hashing ---

func fnSHA256(args []string) (string, error) {
	h := sha256.Sum256([]byte(args[0]))
	return hex.EncodeToString(h[:]), nil
}

func fnSHA384(args []string) (string, error) {
	h := sha512.Sum384([]byte(args[0]))
	return hex.EncodeToString(h[:]), nil
}

func fnSHA512(args []string) (string, error) {
	h := sha512.Sum512([]byte(args[0]))
	return hex.EncodeToString(h[:]), nil
}

// --- Password hashing ---

func fnBcrypt(args []string) (string, error) {
	cost := bcrypt.DefaultCost
	if len(args) >= 2 {
		c, err := strconv.Atoi(args[1])
		if err != nil {
			return "", fmt.Errorf("bcrypt: cost must be an integer: %w", err)
		}
		if c < bcrypt.MinCost || c > bcrypt.MaxCost {
			return "", fmt.Errorf("bcrypt: cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
		}
		cost = c
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(hash), nil
}

// --- Random ---

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func fnRandomString(args []string) (string, error) {
	length, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("random_string: length must be an integer: %w", err)
	}
	if length < 0 || length > 1024 {
		return "", fmt.Errorf("random_string: length must be between 0 and 1024")
	}

	b := make([]byte, length)
	max := big.NewInt(int64(len(alphanumeric)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("random_string: %w", err)
		}
		b[i] = alphanumeric[n.Int64()]
	}
	return string(b), nil
}

func fnRandomHex(args []string) (string, error) {
	byteLen, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("random_hex: length must be an integer: %w", err)
	}
	if byteLen < 0 || byteLen > 512 {
		return "", fmt.Errorf("random_hex: length must be between 0 and 512")
	}

	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random_hex: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// --- String ---

func fnUpper(args []string) (string, error) {
	return strings.ToUpper(args[0]), nil
}

func fnLower(args []string) (string, error) {
	return strings.ToLower(args[0]), nil
}

func fnTrim(args []string) (string, error) {
	return strings.TrimSpace(args[0]), nil
}

func fnReplace(args []string) (string, error) {
	return strings.ReplaceAll(args[0], args[1], args[2]), nil
}

func fnSubstr(args []string) (string, error) {
	s := args[0]
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return "", fmt.Errorf("substr: start must be an integer: %w", err)
	}
	if start < 0 {
		start = 0
	}
	if start > len(s) {
		return "", nil
	}

	if len(args) == 3 {
		length, err := strconv.Atoi(args[2])
		if err != nil {
			return "", fmt.Errorf("substr: length must be an integer: %w", err)
		}
		end := start + length
		if end > len(s) {
			end = len(s)
		}
		return s[start:end], nil
	}

	return s[start:], nil
}

func fnConcat(args []string) (string, error) {
	return strings.Join(args, ""), nil
}

// --- Date/Time ---
//
// Format strings use Go's reference time layout (Mon Jan 2 15:04:05 MST 2006)
// or the following shorthand aliases:
//   rfc3339, iso8601, unix, rfc822, rfc850, rfc1123, kitchen, ansic, stamp

var nowFunc = time.Now // overridable for testing

var formatAliases = map[string]string{
	"rfc3339":   time.RFC3339,
	"iso8601":   time.RFC3339,
	"rfc822":    time.RFC822,
	"rfc850":    time.RFC850,
	"rfc1123":   time.RFC1123,
	"kitchen":   time.Kitchen,
	"ansic":     time.ANSIC,
	"stamp":     time.Stamp,
	"unix":      "unix",
	"unixmilli": "unixmilli",
}

func resolveFormat(args []string, defaultFmt string) string {
	if len(args) == 0 {
		return defaultFmt
	}
	if alias, ok := formatAliases[strings.ToLower(args[0])]; ok {
		return alias
	}
	return args[0]
}

func formatTime(t time.Time, layout string) string {
	switch layout {
	case "unix":
		return strconv.FormatInt(t.Unix(), 10)
	case "unixmilli":
		return strconv.FormatInt(t.UnixMilli(), 10)
	default:
		return t.Format(layout)
	}
}

// date() → "2006-01-02"
// date("02/01/2006") → custom format
func fnDate(args []string) (string, error) {
	layout := resolveFormat(args, "2006-01-02")
	return formatTime(nowFunc().UTC(), layout), nil
}

// time() → "15:04:05"
// time("03:04PM") → custom format
func fnTime(args []string) (string, error) {
	layout := resolveFormat(args, "15:04:05")
	return formatTime(nowFunc().UTC(), layout), nil
}

// datetime() → "2006-01-02T15:04:05Z" (RFC3339)
// datetime("2006/01/02 15:04") → custom format
func fnDatetime(args []string) (string, error) {
	layout := resolveFormat(args, time.RFC3339)
	return formatTime(nowFunc().UTC(), layout), nil
}

// timestamp() → Unix epoch seconds
// timestamp("unixmilli") → Unix epoch milliseconds
// timestamp("rfc3339") → RFC3339 string
func fnTimestamp(args []string) (string, error) {
	layout := resolveFormat(args, "unix")
	return formatTime(nowFunc().UTC(), layout), nil
}
