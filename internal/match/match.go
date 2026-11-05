// Package match canonicalizes manifest symbol IDs so a declaration and an
// extraction finding for the same symbol compare equal even when they disagree
// on the ID form.
//
// The manifest ID conventions (io-surface/v1) allow several forms for one
// symbol:
//
//	pkg.Func            internal/dns.estimatedLines
//	pkg.Type.Method     internal/server.AllowlistManager.Save
//	pkg.Func#aspect     internal/config.Load#file (one symbol, two families)
//	area.topic          logging.sink-topology (no single symbol)
//
// A method that later becomes a free function (or a receiver type rename) must
// not look like a brand-new I/O site: that is noise, and noise is what kills
// adoption. Canonical collapses the receiver: internal/server.Server.Save and
// internal/server.Save both canonicalize to internal/server.Save.
package match

import "strings"

// Canonical returns the receiver-insensitive form of a site ID: the package
// path plus the base symbol name, aspect stripped.
func Canonical(id string) string {
	id = stripAspect(id)
	slash := strings.LastIndex(id, "/")
	head, tail := "", id
	if slash >= 0 {
		head, tail = id[:slash], id[slash+1:]
	}
	// tail is either "file.go"-less dir+symbol ("server.Save") or a bare
	// symbol for root-level packages ("Save"); in both cases the base symbol
	// is the segment after the last '.', with the directory kept intact.
	dot := strings.Index(tail, ".")
	if dot < 0 {
		return id
	}
	dir, symbol := tail[:dot], tail[dot+1:]
	base := symbol
	if i := strings.LastIndex(symbol, "."); i >= 0 {
		base = symbol[i+1:]
	}
	if head == "" {
		return dir + "." + base
	}

	return head + "/" + dir + "." + base
}

// Aspect returns the "#aspect" suffix of an ID, or "".
func Aspect(id string) string {
	if i := strings.Index(id, "#"); i >= 0 {
		return id[i:]
	}

	return ""
}

// stripAspect removes a trailing "#aspect" marker.
func stripAspect(id string) string {
	if before, _, ok := strings.Cut(id, "#"); ok {
		return before
	}

	return id
}

// Package returns the package path portion of an ID (everything before the
// dotted symbol), or "" when the ID has no package.
func Package(id string) string {
	id = stripAspect(id)
	slash := strings.LastIndex(id, "/")
	head, tail := "", id
	if slash >= 0 {
		head, tail = id[:slash], id[slash+1:]
	}
	if dot := strings.Index(tail, "."); dot > 0 {
		if head == "" {
			return tail[:dot]
		}

		return head + "/" + tail[:dot]
	}

	return head
}

// Symbol returns the base symbol name of an ID (no package, no receiver, no
// aspect).
func Symbol(id string) string {
	id = stripAspect(id)
	slash := strings.LastIndex(id, "/")
	tail := id
	if slash >= 0 {
		tail = id[slash+1:]
	}
	if dot := strings.Index(tail, "."); dot > 0 {
		tail = tail[dot+1:]
	}
	if i := strings.LastIndex(tail, "."); i >= 0 {
		tail = tail[i+1:]
	}

	return tail
}
