package gitdata

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/smm-h/reposummary/internal/testutil"
	"github.com/smm-h/reposummary/internal/window"
)

func TestGitBaseNormalRepo(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	base, err := GitBase(repo)
	if err != nil {
		t.Fatalf("GitBase error: %v", err)
	}
	if len(base) != 3 || base[0] != "git" || base[1] != "-C" || base[2] != repo {
		t.Errorf("GitBase = %v, want [git -C %s]", base, repo)
	}
}

func TestGitBaseBare(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())

	// Create a directory containing a bare clone at ".bare".
	container := t.TempDir()
	bare := filepath.Join(container, ".bare")
	if out, err := exec.Command("git", "clone", "--bare", "-q", repo, bare).CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare failed: %v\n%s", err, out)
	}

	base, err := GitBase(container)
	if err != nil {
		t.Fatalf("GitBase error: %v", err)
	}
	if len(base) != 3 || base[0] != "git" || base[1] != "--git-dir" || base[2] != bare {
		t.Errorf("GitBase = %v, want [git --git-dir %s]", base, bare)
	}
}

func TestGitBaseNotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := GitBase(dir); err == nil {
		t.Fatal("GitBase on non-repo should error")
	}
}

func TestCollect(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	base, err := GitBase(repo)
	if err != nil {
		t.Fatalf("GitBase error: %v", err)
	}

	win, err := window.ParseWindow("all")
	if err != nil {
		t.Fatalf("ParseWindow error: %v", err)
	}

	commits, tags, err := Collect(base, win, "HEAD")
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}

	if len(commits) != 5 {
		t.Fatalf("got %d commits, want 5", len(commits))
	}

	// Newest first: last fixture commit ("refactor internals") is first.
	if commits[0].Subject != "refactor internals" {
		t.Errorf("newest subject = %q, want 'refactor internals'", commits[0].Subject)
	}

	for _, c := range commits {
		if len(c.Files) == 0 {
			t.Errorf("commit %q has no files", c.Subject)
		}
		if c.Short == "" || c.Hash == "" {
			t.Errorf("commit %q missing hash/short", c.Subject)
		}
		// The root commit ("feat: add scanner") has 0 parents; the rest have 1.
		wantParents := 1
		if c.Subject == "feat: add scanner" {
			wantParents = 0
		}
		if c.Parents != wantParents {
			t.Errorf("commit %q parents = %d, want %d", c.Subject, c.Parents, wantParents)
		}
	}

	// The breaking commit carries its body.
	var found bool
	for _, c := range commits {
		if c.Subject == "feat!: change API shape" {
			found = true
			if c.Body != "BREAKING CHANGE: removed old flag" {
				t.Errorf("body = %q, want BREAKING CHANGE line", c.Body)
			}
		}
	}
	if !found {
		t.Error("breaking commit not found")
	}

	if len(tags) != 1 || tags[0].Name != "v0.1.0" {
		t.Errorf("tags = %v, want one v0.1.0", tags)
	}
}

func TestCollectEmpty(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	base, err := GitBase(repo)
	if err != nil {
		t.Fatalf("GitBase error: %v", err)
	}
	// A date range far in the future yields no commits.
	win := window.Spec{Mode: "daterange", Since: "2099-01-01", Label: "future"}
	commits, tags, err := Collect(base, win, "HEAD")
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if len(commits) != 0 || len(tags) != 0 {
		t.Errorf("expected empty, got %d commits %d tags", len(commits), len(tags))
	}
}
