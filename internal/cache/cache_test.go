package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ageEntry backdates an entry's mtime so age-based pruning treats it as old.
func ageEntry(t *testing.T, c *Cache, key string, age time.Duration) {
	t.Helper()
	old := time.Now().Add(-age)
	if err := os.Chtimes(c.path(key), old, old); err != nil {
		t.Fatalf("Chtimes(%s): %v", key, err)
	}
}

func TestMakeKeyDeterministic(t *testing.T) {
	a := MakeKey("first", "last", "none", "haiku", "0.1.0", "Last 7 days")
	b := MakeKey("first", "last", "none", "haiku", "0.1.0", "Last 7 days")
	if a != b {
		t.Errorf("MakeKey not deterministic: %q != %q", a, b)
	}
}

func TestMakeKeyDiffers(t *testing.T) {
	base := MakeKey("first", "last", "none", "haiku", "0.1.0", "Last 7 days")
	variants := []string{
		MakeKey("FIRST", "last", "none", "haiku", "0.1.0", "Last 7 days"),
		MakeKey("first", "LAST", "none", "haiku", "0.1.0", "Last 7 days"),
		MakeKey("first", "last", "claude-cli", "haiku", "0.1.0", "Last 7 days"),
		MakeKey("first", "last", "none", "sonnet", "0.1.0", "Last 7 days"),
		MakeKey("first", "last", "none", "haiku", "0.2.0", "Last 7 days"),
		MakeKey("first", "last", "none", "haiku", "0.1.0", "Last 30 days"),
	}
	for i, v := range variants {
		if v == base {
			t.Errorf("variant %d should differ from base", i)
		}
	}
}

// TestMakeKeyEmptyWindowsDiffer guards the zero-commit collapse bug: two
// distinct empty windows (both SHAs empty) must not share a cache key, or a
// cached "no activity" journal could be served under the wrong window heading.
func TestMakeKeyEmptyWindowsDiffer(t *testing.T) {
	today := MakeKey("", "", "none", "haiku", "0.1.0", "Today (2026-07-10)")
	yesterday := MakeKey("", "", "none", "haiku", "0.1.0", "Yesterday (2026-07-09)")
	if today == yesterday {
		t.Errorf("empty windows with different labels collapsed to the same key: %q", today)
	}
}

func TestSetGetRoundtrip(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	key := MakeKey("a", "b", "none", "haiku", "0.1.0", "Last 7 days")
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

// TestSetPrunesAgedEntries verifies that a successful Set opportunistically
// deletes entries older than maxAge while leaving fresh entries alone.
func TestSetPrunesAgedEntries(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	oldKey := MakeKey("old", "old", "none", "haiku", "0.1.0", "Last 7 days")
	freshKey := MakeKey("fresh", "fresh", "none", "haiku", "0.1.0", "Last 7 days")
	if err := c.Set(oldKey, "old journal"); err != nil {
		t.Fatalf("Set(old) error: %v", err)
	}
	if err := c.Set(freshKey, "fresh journal"); err != nil {
		t.Fatalf("Set(fresh) error: %v", err)
	}
	ageEntry(t, c, oldKey, maxAge+24*time.Hour)

	// A subsequent Set triggers the prune pass.
	triggerKey := MakeKey("trigger", "trigger", "none", "haiku", "0.1.0", "Last 7 days")
	if err := c.Set(triggerKey, "trigger journal"); err != nil {
		t.Fatalf("Set(trigger) error: %v", err)
	}

	if _, ok := c.Get(oldKey); ok {
		t.Error("aged entry should have been pruned on Set")
	}
	if _, ok := c.Get(freshKey); !ok {
		t.Error("fresh entry should have survived pruning")
	}
	if _, ok := c.Get(triggerKey); !ok {
		t.Error("just-written entry should be present")
	}
}

// TestGetRefreshesMtimeSurvivesPrune verifies that reading an aged entry
// refreshes its mtime so it survives the next prune pass.
func TestGetRefreshesMtimeSurvivesPrune(t *testing.T) {
	c, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	hotKey := MakeKey("hot", "hot", "none", "haiku", "0.1.0", "Last 7 days")
	if err := c.Set(hotKey, "hot journal"); err != nil {
		t.Fatalf("Set(hot) error: %v", err)
	}
	ageEntry(t, c, hotKey, maxAge+24*time.Hour)

	// Reading the aged entry must refresh its mtime.
	if _, ok := c.Get(hotKey); !ok {
		t.Fatal("Get(hot) missed a key that was just set")
	}

	// A subsequent Set triggers the prune pass; the refreshed entry must survive.
	triggerKey := MakeKey("trigger", "trigger", "none", "haiku", "0.1.0", "Last 7 days")
	if err := c.Set(triggerKey, "trigger journal"); err != nil {
		t.Fatalf("Set(trigger) error: %v", err)
	}
	if _, ok := c.Get(hotKey); !ok {
		t.Error("hot entry read before prune should have survived (mtime refreshed)")
	}
}

// TestSetSurvivesUnprunableContent verifies that non-.md files and
// subdirectories are ignored by the prune pass and Set still succeeds.
func TestSetSurvivesUnprunableContent(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	// An aged non-.md file and an aged subdirectory: both must be left alone.
	sidecar := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(sidecar, []byte("keep me"), 0644); err != nil {
		t.Fatalf("writing sidecar: %v", err)
	}
	subdir := filepath.Join(dir, "sub.md")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	old := time.Now().Add(-maxAge - 24*time.Hour)
	_ = os.Chtimes(sidecar, old, old)
	_ = os.Chtimes(subdir, old, old)

	key := MakeKey("k", "k", "none", "haiku", "0.1.0", "Last 7 days")
	if err := c.Set(key, "journal"); err != nil {
		t.Fatalf("Set should not fail despite unprunable content: %v", err)
	}
	if _, err := os.Stat(sidecar); err != nil {
		t.Error("non-.md sidecar file should be untouched by prune")
	}
	if _, err := os.Stat(subdir); err != nil {
		t.Error("subdirectory should be untouched by prune")
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg-cache-test")
	if got := DefaultDir(); got != "/tmp/xdg-cache-test/reposummary" {
		t.Errorf("DefaultDir = %q, want /tmp/xdg-cache-test/reposummary", got)
	}
}
