package shell

import (
	"testing"
)

func TestRedact(t *testing.T) {
	pat := redactPattern(nil)

	input := []byte(`export PATH=/usr/local/bin:$PATH
export OPENAI_API_KEY=sk-abc123
export HOME=/Users/test
export AWS_SECRET_ACCESS_KEY=mysecret
export GITHUB_TOKEN=ghp_xyz
alias ll="ls -la"
`)
	out, count := redact(input, pat)
	if count != 3 {
		t.Errorf("expected 3 redactions, got %d", count)
	}

	s := string(out)
	if !contains(s, "# REDACTED") {
		t.Error("expected REDACTED placeholder")
	}
	if contains(s, "sk-abc123") {
		t.Error("API key should be redacted")
	}
	if contains(s, "mysecret") {
		t.Error("secret key should be redacted")
	}
	if contains(s, "ghp_xyz") {
		t.Error("token should be redacted")
	}
	// Non-secret exports must survive.
	if !contains(s, "export PATH=/usr/local/bin") {
		t.Error("PATH export should not be redacted")
	}
	if !contains(s, `alias ll="ls -la"`) {
		t.Error("alias should not be redacted")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
