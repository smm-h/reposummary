package main

// Exit codes returned by command handlers.
const (
	exitOK    = 0 // command completed successfully
	exitError = 1 // runtime failure (git error, synthesis error, output error)
	exitUsage = 2 // invalid input: bad flag value or bad repo path
)
