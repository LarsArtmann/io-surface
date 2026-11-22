package main

import "github.com/larsartmann/go-error-family"

// The command-layer sentinels are coded go-error-family rejections: callers
// classify with errors.Is (Is matches code+family) and machines read
// errorfamily.Code. exitCode (main.go) stays the single owner of the 0/1/2
// CLI contract; the family carries semantics, not the exit code.
const (
	codeGateFindings = "gate.findings"
	codeNoSites      = "init.no-sites"
	codeUsage        = "cli.usage"
)

var (
	// errFindings means the gate or an enforcement path found findings; it
	// maps to exit 1. Everything else non-nil exits 2.
	errFindings error = errorfamily.NewRejection(codeGateFindings, "gate findings")
	// errNoSites means extraction found no I/O sites to scaffold.
	errNoSites error = errorfamily.NewRejection(codeNoSites, "no I/O sites found")
	// errUsage means the command's positional arguments are malformed.
	errUsage error = errorfamily.NewRejection(codeUsage, "usage error")
)
