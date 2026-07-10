// Package cache is a deterministic on-disk journal cache. The journal for a
// fixed (firstSHA, lastSHA, synthesis, model, version, windowLabel) tuple is
// deterministic, so identical windows reuse cached output: cost is O(new
// commits), not O(window size). Storage is plain files; no database. Entries
// age out: a successful write prunes entries not read in the last 90 days,
// while a cache hit refreshes an entry's mtime so hot entries stay warm.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxAge is how long a cache entry survives without being read. On each
// successful Set, entries whose mtime is older than this are opportunistically
// pruned; a cache-hit Get refreshes an entry's mtime so hot entries never age
// out.
const maxAge = 90 * 24 * time.Hour

// Cache is a filesystem-backed journal cache rooted at a directory.
type Cache struct {
	dir string
}

// New opens (creating if needed) a cache at dir. An empty dir uses DefaultDir().
func New(dir string) (*Cache, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir}, nil
}

// DefaultDir returns the default cache directory: $XDG_CACHE_HOME/reposummary,
// or ~/.cache/reposummary.
func DefaultDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "reposummary")
}

// MakeKey returns the sha256 hex of the join of the cache inputs. The journal is
// fully determined by this tuple.
//
// windowLabel is part of the key because a zero-commit window has empty
// firstSHA/lastSHA: without the label, every distinct empty window would
// collapse to the same key and a cached "no activity" journal could be served
// under the wrong window heading.
func MakeKey(firstSHA, lastSHA, synthesis, model, version, windowLabel string) string {
	joined := strings.Join([]string{firstSHA, lastSHA, synthesis, model, version, windowLabel}, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

// Get reads a cached journal by key. The second return is false on a miss. On a
// hit the entry's mtime is refreshed so that frequently-read entries survive
// age-based pruning. A failed touch is non-fatal (the read still succeeds).
func (c *Cache) Get(key string) (string, bool) {
	p := c.path(key)
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	now := time.Now()
	_ = os.Chtimes(p, now, now)
	return string(data), true
}

// Set writes a journal to the cache under key, then opportunistically prunes
// entries older than maxAge. Pruning is best-effort housekeeping: its failures
// are swallowed and never turn a successful write into an error.
func (c *Cache) Set(key, md string) error {
	if err := os.WriteFile(c.path(key), []byte(md), 0644); err != nil {
		return err
	}
	c.pruneOld()
	return nil
}

// pruneOld deletes cache entries whose mtime is older than maxAge. Every error
// is ignored: this pass runs after a successful Set and must never break it.
func (c *Cache) pruneOld() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(c.dir, e.Name()))
		}
	}
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key+".md")
}
