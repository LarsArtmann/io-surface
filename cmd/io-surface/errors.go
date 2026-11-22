package main

import "errors"

// The command-layer sentinels. exitCode (main.go) maps them onto the CLI
// contract via errors.Is, never by string matching.
var (
	// errFindings means the gate or an enforcement path found findings; it
	// maps to exit 1. Everything else non-nil exits 2.
	errFindings = errors.New("gate findings")
	// errNoSites means extraction found no I/O sites to scaffold.
	errNoSites = errors.New("no I/O sites found")
	// errUsage means the command's positional arguments are malformed.
	errUsage = errors.New("usage error")
)
