package config

import "testing"

func TestLoadEnabledDefaultsToTrue(t *testing.T) {
	got := LoadEnabled([]string{"diagnostics", "ledger"})
	if !got["diagnostics"] || !got["ledger"] {
		t.Fatalf("LoadEnabled with no env vars set = %v, want all true", got)
	}
}

func TestLoadEnabledRespectsFalse(t *testing.T) {
	t.Setenv("MCP_TOOLS_LEDGER", "false")
	got := LoadEnabled([]string{"diagnostics", "ledger"})
	if !got["diagnostics"] {
		t.Fatalf("diagnostics = %v, want true (unset)", got["diagnostics"])
	}
	if got["ledger"] {
		t.Fatalf("ledger = %v, want false (explicitly disabled)", got["ledger"])
	}
}

func TestLoadEnabledMalformedFailsOpen(t *testing.T) {
	t.Setenv("MCP_TOOLS_LEDGER", "not-a-bool")
	got := LoadEnabled([]string{"ledger"})
	if !got["ledger"] {
		t.Fatalf("ledger = %v, want true (malformed value fails open)", got["ledger"])
	}
}

func TestLoadEnabledCaseInsensitiveDomain(t *testing.T) {
	t.Setenv("MCP_TOOLS_SPREADSHEET", "false")
	got := LoadEnabled([]string{"Spreadsheet"})
	if got["Spreadsheet"] {
		t.Fatalf("Spreadsheet = %v, want false", got["Spreadsheet"])
	}
}
