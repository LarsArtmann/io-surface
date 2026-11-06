// capabilities.cue — reachability facts observed by an external engine.
//
// Written ONLY by `io-surface absorb` (from pinned Capslock output);
// never hand-edit locations, and never let another tool touch this file.
// Advisory tier: capability findings never fail the gate by themselves.
package io_surface

capabilityAudit: {
	engine:  "capslock"
	version: "v0.3.4"
	commit:  "5a6c289-dirty"
	date:    "2026-11-06"
}

capabilities: [
	{pkg: "cmd/io-surface", direct: ["FILES", "NETWORK"], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/absorb", direct: ["EXEC", "FILES"], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/check", direct: ["FILES"], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/emit", direct: [], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/extract", direct: ["FILES", "NETWORK"], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/manifest", direct: ["FILES"], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/render", direct: [], deep: ["EXEC", "FILES", "NETWORK"]},
	{pkg: "internal/scaffold", direct: ["EXEC", "FILES"], deep: ["EXEC", "FILES"]},
]
