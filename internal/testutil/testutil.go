// Package testutil provides shared helpers for building fixture git repos.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// FixtureCommit describes one commit to create in a fixture repo.
type FixtureCommit struct {
	Subject string
	Body    string
	File    string // relative path to create/modify
	Content string // file content
	Date    string // GIT_AUTHOR_DATE/GIT_COMMITTER_DATE, e.g. "2026-01-01T10:00:00"
	Tag     string // lightweight tag to place on this commit ("" = none)
}

// SampleCommits is the canonical fixture used across tests: five commits over
// two days, exercising every classification path, with a lightweight tag.
func SampleCommits() []FixtureCommit {
	return []FixtureCommit{
		{Subject: "feat: add scanner", File: "src/scanner.go", Content: "package src\n", Date: "2026-01-01T10:00:00"},
		{Subject: "fix: correct off-by-one in parser (#12)", File: "src/parser.go", Content: "package src\n// fix\n", Date: "2026-01-01T12:00:00"},
		{Subject: "docs: update readme", File: "README.md", Content: "# readme\n", Date: "2026-01-02T09:00:00"},
		{Subject: "feat!: change API shape", Body: "BREAKING CHANGE: removed old flag", File: "src/api.go", Content: "package src\n// api\n", Date: "2026-01-02T11:00:00", Tag: "v0.1.0"},
		{Subject: "refactor internals", File: "src/util.go", Content: "package src\n// util\n", Date: "2026-01-02T13:00:00"},
	}
}

// BuildFixtureRepo creates a real git repository in a temp directory populated
// with the given commits, and returns its path. Dates are pinned for
// determinism.
func BuildFixtureRepo(t *testing.T, commits []FixtureCommit) string {
	t.Helper()
	dir := t.TempDir()

	run := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run(nil, "init", "-q")
	run(nil, "config", "user.email", "t@t")
	run(nil, "config", "user.name", "Tester")
	run(nil, "config", "commit.gpgsign", "false")

	for _, c := range commits {
		path := filepath.Join(dir, c.File)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(c.Content), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		run(nil, "add", c.File)

		msg := c.Subject
		if c.Body != "" {
			msg += "\n\n" + c.Body
		}
		env := []string{
			"GIT_AUTHOR_DATE=" + c.Date,
			"GIT_COMMITTER_DATE=" + c.Date,
		}
		run(env, "commit", "-q", "-m", msg)

		if c.Tag != "" {
			run(nil, "tag", c.Tag)
		}
	}

	return dir
}
