// Package cache is a deterministic on-disk journal cache. The journal for a
// fixed (firstSHA, lastSHA, synthesis, model, version) tuple is deterministic,
// so identical windows reuse cached output: cost is O(new commits), not
// O(window size). Storage is plain files; no database.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

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
func MakeKey(firstSHA, lastSHA, synthesis, model, version string) string {
	joined := strings.Join([]string{firstSHA, lastSHA, synthesis, model, version}, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

// Get reads a cached journal by key. The second return is false on a miss.
func (c *Cache) Get(key string) (string, bool) {
	data, err := os.ReadFile(c.path(key))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// Set writes a journal to the cache under key.
func (c *Cache) Set(key, md string) error {
	return os.WriteFile(c.path(key), []byte(md), 0644)
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key+".md")
}
