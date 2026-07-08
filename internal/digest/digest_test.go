package digest

import (
	"testing"
	"time"

	"github.com/smm-h/reposummary/internal/gitdata"
	"github.com/smm-h/reposummary/internal/window"
)

func sampleCommits() []gitdata.Commit {
	day1, _ := time.Parse(time.RFC3339, "2026-01-01T10:00:00Z")
	day2, _ := time.Parse(time.RFC3339, "2026-01-02T10:00:00Z")
	return []gitdata.Commit{
		// newest first
		{Hash: "e5", Short: "e5", Author: "Tester", Date: day2, Subject: "refactor internals", Parents: 1,
			Files: []gitdata.FileChange{{Path: "src/util.go", Added: 2, Deleted: 0}}},
		{Hash: "d4", Short: "d4", Author: "Tester", Date: day2, Subject: "feat!: change API shape", Body: "BREAKING CHANGE: removed old flag", Parents: 1,
			Files: []gitdata.FileChange{{Path: "src/api.go", Added: 3, Deleted: 1}}},
		{Hash: "c3", Short: "c3", Author: "Tester", Date: day2, Subject: "docs: update readme", Parents: 1,
			Files: []gitdata.FileChange{{Path: "README.md", Added: 1, Deleted: 0}}},
		{Hash: "b2", Short: "b2", Author: "Tester", Date: day1, Subject: "fix: correct off-by-one in parser (#12)", Parents: 1,
			Files: []gitdata.FileChange{{Path: "src/parser.go", Added: 4, Deleted: 2}}},
		{Hash: "a1", Short: "a1", Author: "Tester", Date: day1, Subject: "feat: add scanner", Parents: 1,
			Files: []gitdata.FileChange{{Path: "src/scanner.go", Added: 10, Deleted: 0}}},
	}
}

func TestClassify(t *testing.T) {
	commits := sampleCommits()
	want := map[string]string{
		"feat: add scanner":                       "feature",
		"fix: correct off-by-one in parser (#12)": "fix",
		"docs: update readme":                     "other",
		"feat!: change API shape":                 "breaking",
		"refactor internals":                      "other",
	}
	for _, c := range commits {
		if got := Classify(c); got != want[c.Subject] {
			t.Errorf("Classify(%q) = %q, want %q", c.Subject, got, want[c.Subject])
		}
	}
}

func TestClassifyNonConventionalFix(t *testing.T) {
	c := gitdata.Commit{Subject: "repair the broken build"}
	if got := Classify(c); got != "fix" {
		t.Errorf("Classify = %q, want fix", got)
	}
}

func TestClassifyNonConventionalFeature(t *testing.T) {
	c := gitdata.Commit{Subject: "introduce new caching layer"}
	if got := Classify(c); got != "feature" {
		t.Errorf("Classify = %q, want feature", got)
	}
}

func TestClassifyBreakingBody(t *testing.T) {
	c := gitdata.Commit{Subject: "rework storage", Body: "BREAKING CHANGE: schema changed"}
	if got := Classify(c); got != "breaking" {
		t.Errorf("Classify = %q, want breaking", got)
	}
}

func TestCleanSubject(t *testing.T) {
	c := gitdata.Commit{Subject: "feat(scope): add scanner"}
	if got := CleanSubject(c); got != "add scanner" {
		t.Errorf("CleanSubject = %q, want 'add scanner'", got)
	}
	c2 := gitdata.Commit{Subject: "refactor internals"}
	if got := CleanSubject(c2); got != "refactor internals" {
		t.Errorf("CleanSubject = %q, want 'refactor internals'", got)
	}
}

func TestBuild(t *testing.T) {
	commits := sampleCommits()
	win, _ := window.ParseWindow("all")
	d := Build(commits, nil, win, "myrepo")

	if d.TotalCommits != 5 {
		t.Errorf("TotalCommits = %d, want 5", d.TotalCommits)
	}
	if d.RepoName != "myrepo" {
		t.Errorf("RepoName = %q, want myrepo", d.RepoName)
	}

	// IssueRefs contains "12".
	foundIssue := false
	for _, r := range d.IssueRefs {
		if r == "12" {
			foundIssue = true
		}
	}
	if !foundIssue {
		t.Errorf("IssueRefs = %v, want to contain 12", d.IssueRefs)
	}

	// DirChurn has "src".
	if d.DirChurn["src"] == 0 {
		t.Errorf("DirChurn missing 'src': %v", d.DirChurn)
	}

	// CommitsByDay has >= 2 keys.
	if len(d.CommitsByDay) < 2 {
		t.Errorf("CommitsByDay = %v, want >= 2 keys", d.CommitsByDay)
	}

	// Buckets classified.
	if len(d.Buckets["feature"]) != 1 {
		t.Errorf("feature bucket = %d, want 1", len(d.Buckets["feature"]))
	}
	if len(d.Buckets["breaking"]) != 1 {
		t.Errorf("breaking bucket = %d, want 1", len(d.Buckets["breaking"]))
	}

	// FirstSHA is oldest (a1), LastSHA is newest (e5).
	if d.FirstSHA != "a1" {
		t.Errorf("FirstSHA = %q, want a1", d.FirstSHA)
	}
	if d.LastSHA != "e5" {
		t.Errorf("LastSHA = %q, want e5", d.LastSHA)
	}

	if !d.HasDates {
		t.Error("HasDates should be true")
	}
}

func TestBuildEmpty(t *testing.T) {
	win, _ := window.ParseWindow("all")
	d := Build(nil, nil, win, "empty")
	if d.HasDates {
		t.Error("HasDates should be false with no commits")
	}
	if d.FirstSHA != "" || d.LastSHA != "" {
		t.Errorf("SHAs should be empty, got %q/%q", d.FirstSHA, d.LastSHA)
	}
}
