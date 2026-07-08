// Package test contains integration tests that build the real reposummary
// binary once and run it as a subprocess against fixture git repositories.
package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/smm-h/reposummary/internal/testutil"
)

var rsBinary string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "reposummary-test-bin-*")
	if err != nil {
		panic("creating temp dir for binary: " + err.Error())
	}

	rsBinary = filepath.Join(tmpDir, "reposummary-test")
	if runtime.GOOS == "windows" {
		rsBinary += ".exe"
	}

	// Project root is two levels up from internal/test/.
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	cmd := exec.Command("go", "build", "-o", rsBinary, ".")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building reposummary binary: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// runRS runs the binary with an isolated HOME and cache dir.
func runRS(t *testing.T, cacheDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(rsBinary, args...)
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") || strings.HasPrefix(e, "XDG_CACHE_HOME=") {
			continue
		}
		env = append(env, e)
	}
	home := t.TempDir()
	env = append(env, "HOME="+home, "XDG_CACHE_HOME="+cacheDir)
	cmd.Env = env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running reposummary: %v", err)
		}
	}
	return stdout, stderr, exitCode
}

func TestSummarizeAll(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	cacheDir := t.TempDir()

	stdout, stderr, code := runRS(t, cacheDir, "summarize", repo, "--window", "all", "--synthesis", "none")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "# Journal") {
		t.Errorf("output missing '# Journal':\n%s", stdout)
	}
	if !strings.Contains(stdout, "## Features") {
		t.Errorf("output missing '## Features':\n%s", stdout)
	}

	// Second run (cache on) yields identical output.
	stdout2, _, code2 := runRS(t, cacheDir, "summarize", repo, "--window", "all", "--synthesis", "none")
	if code2 != 0 {
		t.Fatalf("second run exit %d", code2)
	}
	if stdout != stdout2 {
		t.Errorf("cached run differs:\nfirst:\n%s\nsecond:\n%s", stdout, stdout2)
	}
}

func TestSummarizeNoActivityToday(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	cacheDir := t.TempDir()

	// Fixture commits are dated in 2026-01; "today" has no commits.
	stdout, stderr, code := runRS(t, cacheDir, "summarize", repo, "--window", "today", "--synthesis", "none")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "No activity") {
		t.Errorf("expected 'No activity':\n%s", stdout)
	}
}

func TestSummarizeMissingRequiredFlags(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	cacheDir := t.TempDir()

	// Missing both --window and --synthesis.
	_, _, code := runRS(t, cacheDir, "summarize", repo)
	if code == 0 {
		t.Error("missing required flags should exit nonzero")
	}

	// Missing --synthesis.
	_, _, code = runRS(t, cacheDir, "summarize", repo, "--window", "all")
	if code == 0 {
		t.Error("missing --synthesis should exit nonzero")
	}

	// Missing --window.
	_, _, code = runRS(t, cacheDir, "summarize", repo, "--synthesis", "none")
	if code == 0 {
		t.Error("missing --window should exit nonzero")
	}
}

func TestSummarizeBadWindow(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	cacheDir := t.TempDir()

	_, stderr, code := runRS(t, cacheDir, "summarize", repo, "--window", "nonsense", "--synthesis", "none")
	if code != 2 {
		t.Errorf("bad window exit = %d, want 2 (stderr: %s)", code, stderr)
	}
}

func TestSummarizeNotARepo(t *testing.T) {
	dir := t.TempDir()
	cacheDir := t.TempDir()

	_, stderr, code := runRS(t, cacheDir, "summarize", dir, "--window", "all", "--synthesis", "none")
	if code != 2 {
		t.Errorf("non-repo exit = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "not a git repo") {
		t.Errorf("stderr %q should mention 'not a git repo'", stderr)
	}
}

func TestSummarizeOutputFile(t *testing.T) {
	repo := testutil.BuildFixtureRepo(t, testutil.SampleCommits())
	cacheDir := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "journal.md")

	stdout, stderr, code := runRS(t, cacheDir, "summarize", repo, "--window", "all", "--synthesis", "none", "--output", outFile)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "# Journal") {
		t.Errorf("with --output, stdout should not contain the journal:\n%s", stdout)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), "# Journal") {
		t.Errorf("output file missing journal:\n%s", data)
	}
}
