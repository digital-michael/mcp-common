package modelprefs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func use(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg", "models.json")
	t.Setenv(EnvFile, p)
	return p
}

func TestSaveLoad_roundTripPermissionsAndMissingFile(t *testing.T) {
	path := use(t)
	if p, err := Load(); err != nil || len(p.Aliases) != 0 {
		t.Fatalf("a missing file is empty settings: %+v %v", p, err)
	}
	p := Prefs{}
	if err := p.SetAlias("coding", "qwen2.5-coder:3b", nil); err != nil {
		t.Fatal(err)
	}
	if err := p.SetContext("qwen2.5-coder:3b", 8192); err != nil {
		t.Fatal(err)
	}
	if err := Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || got.Aliases["coding"] != "qwen2.5-coder:3b" || got.Context["qwen2.5-coder:3b"] != 8192 {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v", info.Mode().Perm())
	}
	if m := ResolveFile("coding"); m != "qwen2.5-coder:3b" {
		t.Errorf("ResolveFile: %q", m)
	}
	os.WriteFile(path, []byte("{not json"), 0o600)
	if _, err := Load(); err == nil {
		t.Error("a broken file must be reported")
	}
	if m := ResolveFile("coding"); m != "coding" {
		t.Errorf("a file that cannot be read leaves the name alone: %q", m)
	}
}

func TestAlias_rules(t *testing.T) {
	real := func(n string) bool { return n == "llama3.2:3b" || n == "qwen" }
	p := Prefs{Aliases: map[string]string{"fast": "llama3.2:3b"}}
	for _, tc := range []struct {
		label, target, want string
	}{
		{"coding", "qwen2.5-coder:3b", ""},
		{"Coding", "x", "lower-case"},
		{"has:colon", "x", "lower-case"},
		{"has/slash", "x", "lower-case"},
		{"", "x", "lower-case"},
		{strings.Repeat("a", 41), "x", "lower-case"},
		{"datum-mine", "x", "reserved"},
		{"qwen", "x", "already a model name"},
		{"coding", "", "needs a model"},
		{"coding", "coding", "itself"},
		{"coding", "fast", "itself an alias"},
	} {
		err := p.CheckAlias(tc.label, tc.target, real)
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s -> %s: %v", tc.label, tc.target, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s -> %s: want %q, got %v", tc.label, tc.target, tc.want, err)
		}
	}
	// A label that another alias already points at would create a chain.
	if err := p.CheckAlias("llama3.2:3b", "x", nil); err == nil {
		t.Error("a name with a colon is not an alias label")
	}
	chain := Prefs{Aliases: map[string]string{"a": "b"}}
	if err := chain.CheckAlias("b", "model", nil); err == nil || !strings.Contains(err.Error(), "do not chain") {
		t.Errorf("chain: %v", err)
	}
}

func TestResolveSetRemoveContext(t *testing.T) {
	var p Prefs
	if err := p.SetAlias("coding", "m1", nil); err != nil {
		t.Fatal(err)
	}
	if m, ok := p.Resolve("coding"); m != "m1" || !ok {
		t.Errorf("resolve alias: %q %v", m, ok)
	}
	if m, ok := p.Resolve("m1"); m != "m1" || ok {
		t.Errorf("a real name resolves to itself: %q %v", m, ok)
	}
	p.SetAlias("code2", "m1", nil)
	if got := p.AliasesFor("m1"); len(got) != 2 || got[0] != "code2" {
		t.Errorf("aliases for: %v", got)
	}
	if !p.RemoveAlias("coding") || p.RemoveAlias("coding") {
		t.Error("remove should succeed once")
	}
	for _, bad := range []int{0, 100, MaxContext + 1, -5} {
		if err := p.SetContext("m1", bad); err == nil {
			t.Errorf("accepted context %d", bad)
		}
	}
	if err := p.SetContext("m1", 4096); err != nil {
		t.Fatal(err)
	}
	if n, ok := p.ContextFor("code2"); n != 4096 || !ok {
		t.Errorf("context follows an alias: %d %v", n, ok)
	}
	if _, ok := p.ContextFor("other"); ok {
		t.Error("no context for an unset model")
	}
}
