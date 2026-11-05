package match

import "testing"

func TestCanonical(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"free function", "internal/dns.estimatedLines", "internal/dns.estimatedLines"},
		{"method collapses receiver", "internal/server.AllowlistManager.Save", "internal/server.Save"},
		{"same symbol as function", "internal/server.Save", "internal/server.Save"},
		{"method chain collapses to base", "internal/server.Server.cspReportHandler", "internal/server.cspReportHandler"},
		{"aspect stripped", "internal/config.Load#file", "internal/config.Load"},
		{"aspect stripped from method", "internal/x.T.M#env", "internal/x.M"},
		{"cmd package", "cmd/dnsblockd.loadJSONFile", "cmd/dnsblockd.loadJSONFile"},
		{"nested package dir", "internal/server/sso.Manager.finishLogin", "internal/server/sso.finishLogin"},
		{"area topic without slash", "logging.sink-topology", "logging.sink-topology"},
		{"embed var", "internal/server.openAPISpec", "internal/server.openAPISpec"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Canonical(tc.in); got != tc.want {
				t.Errorf("Canonical(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSymbol(t *testing.T) {
	tests := map[string]string{
		"internal/dns.estimatedLines":           "estimatedLines",
		"internal/server.AllowlistManager.Save": "Save",
		"internal/config.Load#env":              "Load",
		"logging.sink-topology":                 "sink-topology",
	}
	for in, want := range tests {
		if got := Symbol(in); got != want {
			t.Errorf("Symbol(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPackage(t *testing.T) {
	tests := map[string]string{
		"internal/dns.estimatedLines":           "internal/dns",
		"internal/server.AllowlistManager.Save": "internal/server",
		"cmd/dnsblockd.loadJSONFile":            "cmd/dnsblockd",
	}
	for in, want := range tests {
		if got := Package(in); got != want {
			t.Errorf("Package(%q) = %q, want %q", in, got, want)
		}
	}
}
