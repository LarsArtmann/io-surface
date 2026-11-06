// claims.cue — human judgment. Tooling never overwrites this file.
//
// Both files contribute to the same `sites` value (CUE unification), so an entry
// here only needs the claim fields:
//
//	sites: "internal/server.AllowlistManager.Save": claims: {
//		resource:   "temp_allowlist_path JSON"
//		trigger:    "every grant/revoke/expiry sweep/shutdown"
//		discipline: ["atomic-write"]
//		rationale:  "why this posture is right, in prose"
//	}
//
// A decided exception is data, not a comment:
//
//	sites: "internal/dns.timedCacheWrite": claims: {
//		exception: "no-fsync"
//		rationale: "the cache is re-fetched unconditionally after a crash, so a torn file is never parsed"
//	}
package io_surface

sites: "cmd/io-surface.runAbsorb": claims: {
	resource:   "Capslock JSON report passed via --capslock"
	trigger:    "every `io-surface absorb` run"
	discipline: ["bounded"]
	rationale:  "the operator names the artifact; the engine output is decoded in memory with no streaming copy, so an oversized report fails loudly at decode rather than silently truncating"
}

sites: "internal/absorb.ModulePath": claims: {
	resource:   "go.mod of the analyzed repo"
	trigger:    "every absorb run without --module"
	discipline: []
	rationale:  "a single small read to resolve the first-party package prefix; --module exists for manifests analyzed outside their repo"
}

sites: "internal/absorb.gitHead": claims: {
	resource:   "git HEAD + worktree status of the analyzed repo"
	trigger:    "every absorb run without an explicit audit commit"
	discipline: []
	rationale:  "provenance only: the capability audit must name the revision the engine actually analyzed; falls back to the literal working-tree outside git so sandboxed runs stay valid"
}
