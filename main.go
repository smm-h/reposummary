package main

import (
	"github.com/smm-h/strictcli/go/strictcli"
)

func main() {
	app := strictcli.NewApp("reposummary", version,
		"Generate a Markdown journal of a git repository's activity over a time window.")
	registerSummarizeCmd(app)
	app.Run()
}
