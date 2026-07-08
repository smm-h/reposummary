package synth

import (
	"strings"
	"testing"

	"github.com/smm-h/reposummary/internal/digest"
)

func TestSynthesizeNone(t *testing.T) {
	out, err := Synthesize(digest.Digest{}, "none", "haiku")
	if err != nil {
		t.Fatalf("mode none error: %v", err)
	}
	if out != "" {
		t.Errorf("mode none out = %q, want empty", out)
	}
}

func TestSynthesizeAnthropicNoKey(t *testing.T) {
	// Ensure no key is present.
	t.Setenv("ANTHROPIC_API_KEY", "")
	out, err := Synthesize(digest.Digest{}, "anthropic-api", "haiku")
	if err == nil {
		t.Fatal("anthropic-api without key should error")
	}
	if out != "" {
		t.Errorf("out = %q, want empty on error", out)
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("error %q should mention ANTHROPIC_API_KEY", err.Error())
	}
}

func TestSynthesizeUnknownMode(t *testing.T) {
	_, err := Synthesize(digest.Digest{}, "bogus-mode", "haiku")
	if err == nil {
		t.Fatal("unknown mode should error")
	}
	if !strings.Contains(err.Error(), "unknown synthesis mode") {
		t.Errorf("error %q should mention unknown synthesis mode", err.Error())
	}
}
