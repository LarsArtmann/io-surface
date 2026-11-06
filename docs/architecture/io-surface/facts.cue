// facts.cue — extracted from code by `io-surface init`.
//
// Mechanical file: regenerate with `io-surface init --force` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

audit: {
	commit: "4cc7524-dirty"
	date:   "2026-11-06"
	claims: 15
	method: "io-surface extract (AST, stdlib-only)"
}

scope: {
	in: [
		"file system (reads, writes, embeds)",
		"databases (SQLite/engine handles)",
		"network listeners",
		"outbound network",
		"stream copies",
		"entropy",
		"subprocess / local IPC",
		"environment reads",
	]
	out: [
		"timers and tickers (scheduling)",
		"memory pooling",
		"kernel-internal I/O (epoll, page cache)",
	]
}

sites: {
	"cmd/io-surface.applyDoc": {
		location: [
			"cmd/io-surface/commands.go:165",
			"cmd/io-surface/commands.go:201",
		]
		family: "file-write"
		facts: {
			call: ["os.ReadFile", "os.WriteFile"]
		}
	}
	"cmd/io-surface.runRender": {
		location: [
			"cmd/io-surface/commands.go:158",
		]
		family: "file-write"
		facts: {
			call: ["os.WriteFile"]
		}
	}
	"internal/check.symbolVisible": {
		location: [
			"internal/check/location.go:23",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.Extract": {
		location: [
			"internal/extract/extract.go:111",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/extract.extractor.parseFile": {
		location: [
			"internal/extract/extract.go:276",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.walk": {
		location: [
			"internal/extract/extract.go:237",
		]
		family: "file-read"
		facts: {
			call: ["filepath.Walk"]
		}
	}
	"internal/manifest.Load": {
		location: [
			"internal/manifest/manifest.go:121",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/manifest.Manifest.Validate": {
		location: [
			"internal/manifest/manifest.go:297",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/scaffold.Write": {
		location: [
			"internal/scaffold/scaffold.go:55",
			"internal/scaffold/scaffold.go:58",
			"internal/scaffold/scaffold.go:70",
			"internal/scaffold/scaffold.go:74",
		]
		family: "file-write"
		facts: {
			call: ["os.MkdirAll", "os.Stat", "os.WriteFile"]
		}
	}
	"internal/scaffold.git": {
		location: [
			"internal/scaffold/scaffold.go:205",
		]
		family: "ipc-subprocess"
		facts: {
			call: ["exec.Command"]
		}
	}
	"internal/scaffold.schemaCUE": {
		location: [
			"internal/scaffold/scaffold.go:30",
		]
		family: "embed"
		facts: {
			call: ["go:embed schema.cue"]
		}
	}
}
