package cache

import (
	"testing"
)

func TestMakeKeyDeterministic(t *testing.T) {
	a := MakeKey("first", "last", "none", "haiku", "0.1.0")
	b := MakeKey("first", "last", "none", "haiku", "0.1.0")
	if a != b {
		t.Errorf("MakeKey not deterministic: %q != %q", a, b)
	}
}

func TestMakeKeyDiffers(t *testing.T) {
	base := MakeKey("first", "last", "none", "haiku", "0.1.0")
	variants := []string{
		MakeKey("FIRST", "last", "none", "haiku", "0.1.0"),
		MakeKey("first", "LAST", "none", "haiku", "0.1.0"),
		MakeKey("first", "last", "claude-cli", "haiku", "0.1.0"),
		MakeKey("first", "last", "none", "sonnet", "0.1.0"),
		MakeKey("first", "last", "none", "haiku", "0.2.0"),
	}
	for i, v := range variants {
		if v == base {
			t.Errorf("variant %d should differ from base", i)
		}
	}
}

func TestSetGetRoundtrip(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	key := MakeKey("a", "b", "none", "haiku", "0.1.0")
	content := "# Journal\n\nsome markdown"
	if err := c.Set(key, content); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	got, ok := c.Get(key)
	if !ok {
		t.Fatal("Get missed a key that was just set")
	}
	if got != content {
		t.Errorf("Get = %q, want %q", got, content)
	}
}

func TestGetMissing(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	got, ok := c.Get("nonexistent-key")
	if ok {
		t.Errorf("Get(missing) ok = true, want false")
	}
	if got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache-test")
	if got := DefaultDir(); got != "/tmp/xdg-cache-test/reposummary" {
		t.Errorf("DefaultDir = %q, want /tmp/xdg-cache-test/reposummary", got)
	}
}
