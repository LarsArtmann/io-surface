// facts.cue — extracted from code by `io-surface init`.
//
// Mechanical file: regenerate with `io-surface init --force` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

audit: {
	commit: "e803210-dirty"
	date:   "2026-11-07"
	claims: 24
	method: "io-surface extract (AST, stdlib-only)"
}

enforcement: "gate"

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
			"cmd/io-surface/commands.go:300",
			"cmd/io-surface/commands.go:336",
		]
		family: "file-write"
		facts: {
			call: ["os.ReadFile", "os.WriteFile"]
		}
	}
	"cmd/io-surface.runAbsorb": {
		location: [
			"cmd/io-surface/commands.go:214",
			"cmd/io-surface/commands.go:243",
		]
		family: "file-write"
		facts: {
			call: ["os.ReadFile", "os.WriteFile"]
		}
	}
	"cmd/io-surface.runRender": {
		location: [
			"cmd/io-surface/commands.go:293",
		]
		family: "file-write"
		facts: {
			call: ["os.WriteFile"]
		}
	}
	"internal/absorb.ModulePath": {
		location: [
			"internal/absorb/absorb.go:78",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/absorb.gitHead": {
		location: [
			"internal/absorb/absorb.go:165",
			"internal/absorb/absorb.go:172",
		]
		family: "ipc-subprocess"
		facts: {
			call: ["exec.Command"]
		}
	}
	"internal/check.symbolVisible": {
		location: [
			"internal/check/location.go:28",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.collectSeamDecl": {
		location: [
			"internal/extract/extract.go:167",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.parseFile": {
		location: [
			"internal/extract/extract.go:230",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.walk": {
		location: [
			"internal/extract/extract.go:90",
		]
		family: "file-read"
		facts: {
			call: ["filepath.Walk"]
		}
	}
	"internal/extract.runExtract": {
		location: [
			"internal/extract/tables.go:106",
			"internal/extract/tables.go:96",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/manifest.AppendBaseline": {
		location: [
			"internal/manifest/baseline_write.go:39",
			"internal/manifest/baseline_write.go:59",
		]
		family: "file-write"
		facts: {
			call: ["os.ReadFile", "os.WriteFile"]
		}
	}
	"internal/manifest.Load": {
		location: [
			"internal/manifest/manifest.go:192",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/manifest.Manifest.Validate": {
		location: [
			"internal/manifest/manifest.go:402",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/scaffold.Write": {
		location: [
			"internal/scaffold/scaffold.go:59",
			"internal/scaffold/scaffold.go:62",
			"internal/scaffold/scaffold.go:74",
			"internal/scaffold/scaffold.go:78",
		]
		family: "file-write"
		facts: {
			call: ["os.MkdirAll", "os.Stat", "os.WriteFile"]
		}
	}
	"internal/scaffold.git": {
		location: [
			"internal/scaffold/scaffold.go:226",
		]
		family: "ipc-subprocess"
		facts: {
			call: ["exec.Command"]
		}
	}
	"internal/scaffold.schemaCUE": {
		location: [
			"internal/scaffold/scaffold.go:31",
		]
		family: "embed"
		facts: {
			call: ["go:embed schema.cue"]
		}
	}
}
