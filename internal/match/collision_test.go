package match

import "testing"

func TestCanonical_KnownCollisionNeedsExactPairing(t *testing.T) {
	// Two different types in one package with the same method name collapse to
	// ONE canonical key. That is deliberate (receiver renames must not look like
	// new sites), and it is why the checker pairs exact IDs first and only then
	// falls back to canonical pairing: a canonical key alone cannot tell these
	// two symbols apart.
	a := Canonical("internal/server.AllowlistManager.Load")
	b := Canonical("internal/server.PermanentAllowlist.Load")
	if a != b {
		t.Fatalf("expected the documented collision, got %q and %q", a, b)
	}
	if a != "internal/server.Load" {
		t.Errorf("canonical = %q, want internal/server.Load", a)
	}
}
