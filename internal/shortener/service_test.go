package shortener

import (
	"testing"
)

func TestGenerateCode(t *testing.T) {
	code := generateCode()
	if len(code) != codeLength {
		t.Errorf("expected code length %d, got %d", codeLength, len(code))
	}

	// All characters should be base62
	for _, c := range code {
		found := false
		for _, b := range base62Chars {
			if c == b {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("code contains non-base62 character: %c", c)
		}
	}
}

func TestGenerateCodeUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		code := generateCode()
		if seen[code] {
			t.Errorf("duplicate code generated: %s", code)
		}
		seen[code] = true
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com/path?q=1", false},
		{"valid with port", "https://example.com:8080/path", false},
		{"empty", "", true},
		{"no scheme", "example.com", true},
		{"ftp scheme", "ftp://example.com", true},
		{"javascript", "javascript:alert(1)", true},
		{"data uri", "data:text/html,<h1>hi</h1>", true},
		{"no host", "http://", true},
		{"relative path", "/path/to/thing", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestAliasValidation(t *testing.T) {
	tests := []struct {
		name  string
		alias string
		valid bool
	}{
		{"simple", "abc123", true},
		{"single char", "a", true},
		{"max length 12", "abcABC123456", true},
		{"too long 13", "abcABC1234567", false},
		{"empty", "", false},
		{"with dash", "my-link", false},
		{"with underscore", "my_link", false},
		{"with slash", "../../etc", false},
		{"with space", "my link", false},
		{"html", "<script>", false},
		{"emoji", "\U0001f680", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aliasRegex.MatchString(tt.alias)
			if got != tt.valid {
				t.Errorf("aliasRegex.MatchString(%q) = %v, want %v", tt.alias, got, tt.valid)
			}
		})
	}
}

func TestIsDuplicateKeyErr(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"ERROR: duplicate key value violates unique constraint", true},
		{"pq: error code 23505", true},
		{"connection refused", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := &testError{msg: tt.msg}
			if got := isDuplicateKeyErr(err); got != tt.want {
				t.Errorf("isDuplicateKeyErr(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
