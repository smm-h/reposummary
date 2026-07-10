package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/smm-h/reposummary/internal/cache"
	"github.com/smm-h/reposummary/internal/digest"
	"github.com/smm-h/reposummary/internal/gitdata"
	"github.com/smm-h/reposummary/internal/render"
	"github.com/smm-h/reposummary/internal/synth"
	"github.com/smm-h/reposummary/internal/window"
	"github.com/smm-h/strictcli/go/strictcli"
)

func registerSummarizeCmd(app *strictcli.App) {
	app.Command("summarize",
		"Extract a git repository's activity over a time window and render a Markdown journal, optionally narrated by an LLM.",
		handleSummarize,
		strictcli.WithFlags(
			strictcli.StringFlag("window",
				"time window: today | yesterday | week | month | all | <YYYY-MM-DD>+<N><unit> | <refA>..<refB>"),
			strictcli.StringFlag("synthesis",
				"LLM synthesis backend for the prose summary: none, claude-cli, or anthropic-api (no silent fallback)",
				strictcli.Choices("none", "claude-cli", "anthropic-api")),
			strictcli.StringFlag("model",
				"model id for LLM synthesis (e.g. haiku); used only when --synthesis is claude-cli or anthropic-api",
				strictcli.Default("haiku")),
			strictcli.StringFlag("branch",
				"git ref to treat as the tip of history when resolving the window (defaults to HEAD)",
				strictcli.Default("HEAD")),
			strictcli.StringFlag("output",
				"write the generated Markdown journal to this file instead of printing to standard output",
				strictcli.Default("")),
			strictcli.StringFlag("cache-dir",
				"override the directory used for the on-disk journal cache (defaults to the user cache dir)",
				strictcli.Default("")),
			strictcli.BoolFlag("cache",
				"use the on-disk journal cache; pass --no-cache to always recompute the journal from scratch",
				strictcli.Default(true)),
		),
		strictcli.WithArgs(
			strictcli.NewArg("repo", "path to the git repository to summarize (a normal checkout or a bare + worktree layout)",
				strictcli.ArgRequired(false), strictcli.ArgDefault(".")),
		),
	)
}

// errorf prints a handler error line to stderr with the "error:" prefix.
func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

func handleSummarize(kwargs map[string]interface{}) int {
	repo := kwargs["repo"].(string)
	synthesis := kwargs["synthesis"].(string)
	model := kwargs["model"].(string)
	tip := kwargs["branch"].(string)
	output := kwargs["output"].(string)
	cacheDir := kwargs["cache_dir"].(string)
	useCache := kwargs["cache"].(bool)

	base, err := gitdata.GitBase(repo)
	if err != nil {
		errorf("%s", err)
		return exitUsage
	}

	win, err := window.ParseWindow(kwargs["window"].(string))
	if err != nil {
		errorf("%s", err)
		return exitUsage
	}

	commits, tags, err := gitdata.Collect(base, win, tip)
	if err != nil {
		errorf("%s", err)
		return exitError
	}

	absRepo, err := filepath.Abs(repo)
	if err != nil {
		errorf("resolving repo path %q: %s", repo, err)
		return exitUsage
	}
	dig := digest.Build(commits, tags, win, filepath.Base(absRepo))

	var c *cache.Cache
	var key string
	if useCache {
		c, err = cache.New(cacheDir)
		if err != nil {
			errorf("opening cache: %s", err)
			return exitError
		}
		key = cache.MakeKey(dig.FirstSHA, dig.LastSHA, synthesis, model, version, win.Label)
		if md, ok := c.Get(key); ok {
			return writeOutput(md, output)
		}
	}

	narrative, err := synth.Synthesize(dig, synthesis, model)
	if err != nil {
		errorf("%s", err)
		return exitError
	}

	md := render.Markdown(dig, narrative, version)

	if useCache {
		if err := c.Set(key, md); err != nil {
			// Cache write failure is non-fatal, but not swallowed silently.
			fmt.Fprintf(os.Stderr, "warning: writing cache entry failed: %s\n", err)
		}
	}

	return writeOutput(md, output)
}

// writeOutput writes the journal to a file or stdout.
func writeOutput(md, output string) int {
	if output != "" {
		if err := os.WriteFile(output, []byte(md), 0644); err != nil {
			errorf("writing output file %q: %s", output, err)
			return exitError
		}
		return exitOK
	}
	fmt.Println(md)
	return exitOK
}
