// Package gitdata extracts commit metadata, file churn, and tags from a git
// repository over a resolved window. Everything is read deterministically from
// git subprocesses; no LLM tokens are spent here.
package gitdata

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/smm-h/reposummary/internal/window"
)

// FileChange is a single file touched by a commit. Added/Deleted are 0 for
// binary files (git reports "-" for those).
type FileChange struct {
	Path    string
	Added   int
	Deleted int
}

// Commit is a single commit's extracted metadata.
type Commit struct {
	Hash    string
	Short   string
	Author  string
	Date    time.Time
	Subject string
	Body    string
	Parents int
	Files   []FileChange
}

// Tag is a git tag whose target resolves to a collected commit.
type Tag struct {
	Name string
	Hash string
}

// GitBase returns the git command prefix for a repo, or an error if the path is
// not a git repository. It supports normal repos (.git dir or a working git
// rev-parse) and the ".bare" convention (a bare repo stored under <repo>/.bare).
func GitBase(repo string) ([]string, error) {
	gitDir := filepath.Join(repo, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return []string{"git", "-C", repo}, nil
	}
	// A working `git rev-parse --git-dir` (exit 0) means this is a repo, even
	// when .git is a file (worktrees) or the cwd is inside the work tree.
	cmd := exec.Command("git", "-C", repo, "rev-parse", "--git-dir")
	if err := cmd.Run(); err == nil {
		return []string{"git", "-C", repo}, nil
	}
	bare := filepath.Join(repo, ".bare")
	if info, err := os.Stat(bare); err == nil && info.IsDir() {
		return []string{"git", "--git-dir", bare}, nil
	}
	return nil, fmt.Errorf("not a git repo: %s", repo)
}

// selector builds the git log selector args for a window and tip.
func selector(win window.Spec, tip string) []string {
	var sel []string
	if win.RevRange != "" {
		sel = append(sel, win.RevRange)
	} else {
		sel = append(sel, tip)
		if win.Since != "" {
			sel = append(sel, "--since="+win.Since)
		}
		if win.Until != "" {
			sel = append(sel, "--until="+win.Until)
		}
	}
	return sel
}

// runGit runs a git command built from base + args and returns stdout. A
// non-zero exit becomes an error that includes stderr.
func runGit(base, args []string) (string, error) {
	full := append(append([]string{}, base[1:]...), args...)
	cmd := exec.Command(base[0], full...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

// Collect gathers commits, their file churn, and matching tags for a window.
// Empty output yields empty slices and a nil error.
func Collect(base []string, win window.Spec, tip string) ([]Commit, []Tag, error) {
	sel := selector(win, tip)

	commits, err := collectMeta(base, sel)
	if err != nil {
		return nil, nil, err
	}
	if len(commits) == 0 {
		return nil, nil, nil
	}

	if err := attachNumstat(base, sel, commits); err != nil {
		return nil, nil, err
	}

	tags, err := collectTags(base, commits)
	if err != nil {
		return nil, nil, err
	}

	return commits, tags, nil
}

// collectMeta runs pass 1: metadata for every commit in git order (newest first).
func collectMeta(base, sel []string) ([]Commit, error) {
	args := append([]string{"log"}, sel...)
	args = append(args, "--no-color", "--date=iso-strict",
		"--pretty=format:%H%x1f%h%x1f%an%x1f%aI%x1f%P%x1f%s%x1f%b%x1e")

	out, err := runGit(base, args)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var commits []Commit
	records := strings.Split(out, "\x1e")
	for _, rec := range records {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		fields := strings.Split(rec, "\x1f")
		if len(fields) < 7 {
			continue
		}
		date, err := time.Parse(time.RFC3339, strings.TrimSpace(fields[3]))
		if err != nil {
			return nil, fmt.Errorf("parsing commit date %q: %w", fields[3], err)
		}
		parents := 0
		if p := strings.TrimSpace(fields[4]); p != "" {
			parents = len(strings.Fields(p))
		}
		commits = append(commits, Commit{
			Hash:    strings.TrimSpace(fields[0]),
			Short:   strings.TrimSpace(fields[1]),
			Author:  fields[2],
			Date:    date,
			Subject: fields[5],
			Body:    strings.TrimRight(fields[6], "\n"),
			Parents: parents,
		})
	}
	return commits, nil
}

// attachNumstat runs pass 2: numstat, attaching FileChanges to commits by hash.
func attachNumstat(base, sel []string, commits []Commit) error {
	args := append([]string{"log"}, sel...)
	args = append(args, "--no-color", "--numstat", "--pretty=format:%x1e%H")

	out, err := runGit(base, args)
	if err != nil {
		return err
	}

	byHash := make(map[string]*Commit, len(commits))
	for i := range commits {
		byHash[commits[i].Hash] = &commits[i]
	}

	var current *Commit
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "\x1e") {
			hash := strings.TrimSpace(strings.TrimPrefix(line, "\x1e"))
			current = byHash[hash]
			continue
		}
		if current == nil || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		added := parseNumstat(parts[0])
		deleted := parseNumstat(parts[1])
		current.Files = append(current.Files, FileChange{
			Path:    parts[2],
			Added:   added,
			Deleted: deleted,
		})
	}
	return nil
}

// parseNumstat parses a numstat count; "-" (binary) becomes 0.
func parseNumstat(s string) int {
	s = strings.TrimSpace(s)
	if s == "-" || s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// collectTags returns a Tag for every tag whose resolved target is one of the
// collected commits.
func collectTags(base []string, commits []Commit) ([]Tag, error) {
	hashes := make(map[string]bool, len(commits))
	for _, c := range commits {
		hashes[c.Hash] = true
	}

	// git for-each-ref (unlike git log) does not interpret %x1f as a hex
	// escape, so we embed the actual 0x1f separator byte in the format.
	args := []string{"for-each-ref",
		"--format=%(refname:short)\x1f%(objectname)\x1f%(*objectname)", "refs/tags"}
	out, err := runGit(base, args)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\x1f")
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		objectName := strings.TrimSpace(fields[1])
		deref := strings.TrimSpace(fields[2])
		resolved := objectName
		if deref != "" {
			resolved = deref
		}
		if hashes[resolved] {
			tags = append(tags, Tag{Name: name, Hash: resolved})
		}
	}
	return tags, nil
}
