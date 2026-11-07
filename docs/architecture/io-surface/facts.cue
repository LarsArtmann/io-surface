// facts.cue — extracted from code by `io-surface init`.
//
// Mechanical file: regenerate with `io-surface init --force` after an
// I/O-adding change; never hand-edit locations. Judgment lives in claims.cue.
package io_surface

audit: {
	commit: "77ba452-dirty"
	date:   "2026-11-07"
	claims: 23
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
			"cmd/io-surface/commands.go:301",
			"cmd/io-surface/commands.go:338",
		]
		family: "file-write"
		facts: {
			call: ["atomicwrite.WriteVerified", "os.ReadFile"]
		}
	}
	"cmd/io-surface.runAbsorb": {
		location: [
			"cmd/io-surface/commands.go:215",
			"cmd/io-surface/commands.go:244",
		]
		family: "file-write"
		facts: {
			call: ["atomicwrite.WriteWithPerm", "os.ReadFile"]
		}
	}
	"cmd/io-surface.runRender": {
		location: [
			"cmd/io-surface/commands.go:294",
		]
		family: "file-write"
		facts: {
			call: ["atomicwrite.WriteWithPerm"]
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
			"internal/extract/extract.go:171",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.parseFile": {
		location: [
			"internal/extract/extract.go:234",
		]
		family: "file-read"
		facts: {
			call: ["os.ReadFile"]
		}
	}
	"internal/extract.extractor.walk": {
		location: [
			"internal/extract/extract.go:94",
		]
		family: "file-read"
		facts: {
			call: ["filepath.Walk"]
		}
	}
	"internal/extract.runExtract": {
		location: [
			"internal/extract/tables.go:107",
			"internal/extract/tables.go:97",
		]
		family: "file-read"
		facts: {
			call: ["os.Stat"]
		}
	}
	"internal/manifest.AppendBaseline": {
		location: [
			"internal/manifest/baseline_write.go:41",
			"internal/manifest/baseline_write.go:62",
		]
		family: "file-write"
		facts: {
			call: ["atomicwrite.WriteVerified", "os.ReadFile"]
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
			"internal/scaffold/scaffold.go:61",
			"internal/scaffold/scaffold.go:64",
			"internal/scaffold/scaffold.go:76",
		]
		family: "file-write"
		facts: {
			call: ["os.MkdirAll", "os.Stat"]
		}
	}
	"internal/scaffold.git": {
		location: [
			"internal/scaffold/scaffold.go:228",
		]
		family: "ipc-subprocess"
		facts: {
			call: ["exec.Command"]
		}
	}
	"internal/scaffold.schemaCUE": {
		location: [
			"internal/scaffold/scaffold.go:33",
		]
		family: "embed"
		facts: {
			call: ["go:embed schema.cue"]
		}
	}
}
