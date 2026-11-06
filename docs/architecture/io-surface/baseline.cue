// baseline.cue — grandfathered undeclared sites (the adoption escape hatch).
//
// The baseline only shrinks: every entry needs a note and a date, and a new
// entry needs a reason in the commit message. Never use it for a site you are
// about to fix "later" without a date.
package io_surface

baseline: [
	{
		id:   "cmd/io-surface"
		note: "capslock v0.3.4 direct-NETWORK edge here is an SSA conflation phantom (json/v2 stream writer unified with http.response; zero network use — blind-spot run 2026-11-06, false-positive class 2)"
		date: "2026-11-06"
	},
	{
		id:   "internal/extract"
		note: "capslock v0.3.4 direct-NETWORK edges here are SSA range-func conflation (strings.FieldsSeq$1 -> net/http.ParseCookie$1; the package is hermetic by contract — blind-spot run 2026-11-06, false-positive class 1)"
		date: "2026-11-06"
	},
]
