// Package digest turns raw git data into a structured, deterministic summary:
// commit classification, per-directory churn, issue references, author counts,
// and per-day activity. This is the free (no-LLM) narrative layer.
package digest

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/smm-h/reposummary/internal/gitdata"
	"github.com/smm-h/reposummary/internal/window"
)

// conventionalRE matches a Conventional Commits subject prefix, e.g.
// "feat(scope)!: ...". Groups: 1=type, 3=scope, 4=bang.
var conventionalRE = regexp.MustCompile(`^(\w+)(\(([^)]+)\))?(!)?:\s`)

var fixRE = regexp.MustCompile(`(?i)\b(fix|bug|bugfix|repair|correct|resolve)\b`)
var featRE = regexp.MustCompile(`(?i)\b(add|implement|introduce|support|new|create)\b`)
var issueRE = regexp.MustCompile(`#(\d+)`)

// Classify returns the change category for a commit: one of "breaking",
// "feature", "fix", or "other".
func Classify(c gitdata.Commit) string {
	if m := conventionalRE.FindStringSubmatch(c.Subject); m != nil {
		typ := strings.ToLower(m[1])
		bang := m[4] == "!"
		if bang || strings.Contains(c.Body, "BREAKING CHANGE") {
			return "breaking"
		}
		switch typ {
		case "feat", "perf":
			return "feature"
		case "fix":
			return "fix"
		default:
			// docs, chore, refactor, test, build, ci, style, unknown
			return "other"
		}
	}

	// Non-conventional subject.
	if strings.Contains(c.Body, "BREAKING CHANGE") {
		return "breaking"
	}
	if fixRE.MatchString(c.Subject) {
		return "fix"
	}
	if featRE.MatchString(c.Subject) {
		return "feature"
	}
	return "other"
}

// CleanSubject strips a Conventional Commits prefix if present, otherwise
// returns the subject unchanged. The result is trimmed.
func CleanSubject(c gitdata.Commit) string {
	if loc := conventionalRE.FindStringIndex(c.Subject); loc != nil {
		return strings.TrimSpace(c.Subject[loc[1]:])
	}
	return strings.TrimSpace(c.Subject)
}

// Digest is the structured summary of a window's activity.
type Digest struct {
	RepoName    string
	WindowLabel string
	DateStart   time.Time // zero value if empty
	DateEnd     time.Time
	HasDates    bool

	TotalCommits int
	TotalFiles   int
	Insertions   int
	Deletions    int
	Merges       int

	Authors map[string]int
	Buckets map[string][]gitdata.Commit // keys: breaking, feature, fix, other
	Tags    []gitdata.Tag

	DirChurn     map[string]int // top-level dir -> touches
	IssueRefs    []string       // sorted unique issue numbers
	CommitsByDay map[string]int // YYYY-MM-DD -> count

	FirstSHA string // oldest commit hash ("" if empty)
	LastSHA  string // newest commit hash ("" if empty)
}

// Build assembles a Digest from collected commits and tags. Commits are assumed
// to be newest-first (git log order).
func Build(commits []gitdata.Commit, tags []gitdata.Tag, win window.Spec, repoName string) Digest {
	d := Digest{
		RepoName:     repoName,
		WindowLabel:  win.Label,
		TotalCommits: len(commits),
		Authors:      make(map[string]int),
		Buckets:      map[string][]gitdata.Commit{},
		Tags:         tags,
		DirChurn:     make(map[string]int),
		CommitsByDay: make(map[string]int),
	}

	issues := make(map[string]bool)

	for _, c := range commits {
		d.Authors[c.Author]++
		if c.Parents > 1 {
			d.Merges++
		}
		d.TotalFiles += len(c.Files)
		for _, f := range c.Files {
			d.Insertions += f.Added
			d.Deletions += f.Deleted
			top := topLevelDir(f.Path)
			d.DirChurn[top]++
		}

		cat := Classify(c)
		d.Buckets[cat] = append(d.Buckets[cat], c)

		for _, m := range issueRE.FindAllStringSubmatch(c.Subject+"\n"+c.Body, -1) {
			issues[m[1]] = true
		}

		day := c.Date.Format("2006-01-02")
		d.CommitsByDay[day]++

		if !d.HasDates {
			d.DateStart = c.Date
			d.DateEnd = c.Date
			d.HasDates = true
		} else {
			if c.Date.Before(d.DateStart) {
				d.DateStart = c.Date
			}
			if c.Date.After(d.DateEnd) {
				d.DateEnd = c.Date
			}
		}
	}

	// Issue refs: unique, sorted numerically.
	for k := range issues {
		d.IssueRefs = append(d.IssueRefs, k)
	}
	sort.Slice(d.IssueRefs, func(i, j int) bool {
		ai, _ := strconv.Atoi(d.IssueRefs[i])
		aj, _ := strconv.Atoi(d.IssueRefs[j])
		return ai < aj
	})

	if len(commits) > 0 {
		d.LastSHA = commits[0].Hash               // newest
		d.FirstSHA = commits[len(commits)-1].Hash // oldest
	}

	return d
}

// topLevelDir returns the first path segment before "/", or the path itself if
// it has no directory component.
func topLevelDir(path string) string {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}
