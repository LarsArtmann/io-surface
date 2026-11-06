// facts.cue — extracted from code by `io-surface init`.
//
// Mechanical file: regenerate with `io-surface init --force` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

audit: {
	commit: "working-tree"
	date:   "2026-11-06"
	claims: 14
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
			"cmd/io-surface/main.go:261",
			"cmd/io-surface/main.go:297",
		]
		family: "file-write"
		facts: {
			call: ["os.ReadFile", "os.WriteFile"]
		}
	}
	"cmd/io-surface.runRender": {
		location: [
			"cmd/io-surface/main.go:254",
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
			"internal/manifest/manifest.go:116",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/manifest.Manifest.Validate": {
		location: [
			"internal/manifest/manifest.go:190",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/scaffold.Write": {
		location: [
			"internal/scaffold/scaffold.go:53",
			"internal/scaffold/scaffold.go:56",
			"internal/scaffold/scaffold.go:68",
			"internal/scaffold/scaffold.go:72",
		]
		family: "file-write"
		facts: {
			call: ["os.MkdirAll", "os.Stat", "os.WriteFile"]
		}
	}
	"internal/scaffold.schemaCUE": {
		location: [
			"internal/scaffold/scaffold.go:29",
		]
		family: "embed"
		facts: {
			call: ["go:embed schema.cue"]
		}
	}
}
