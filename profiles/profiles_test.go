package profiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dir(t *testing.T) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), "profiles")
	t.Setenv(EnvDir, d)
	return d
}

func TestSaveLoad_roundTripMergeAndPermissions(t *testing.T) {
	d := dir(t)
	if err := Save("p", map[string]string{"a": "1", "b": "2"}); err != nil {
		t.Fatal(err)
	}
	if err := Save("p", map[string]string{"b": "", "c": "3"}); err != nil {
		t.Fatal(err)
	}
	got, err := Load("p")
	if err != nil || len(got) != 2 || got["a"] != "1" || got["c"] != "3" {
		t.Errorf("merge: %v %v", got, err)
	}
	if info, _ := os.Stat(filepath.Join(d, "p.json")); info.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v", info.Mode().Perm())
	}
	if info, _ := os.Stat(d); info.Mode().Perm() != 0o700 {
		t.Errorf("dir mode %v", info.Mode().Perm())
	}
	if m, err := Load("absent"); err != nil || len(m) != 0 {
		t.Errorf("a missing profile is empty, not an error: %v %v", m, err)
	}
}

func TestLoad_refusesAReadableFileAndBadNames(t *testing.T) {
	d := dir(t)
	Save("p", map[string]string{"a": "1"})
	os.Chmod(filepath.Join(d, "p.json"), 0o644)
	if _, err := Load("p"); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("a world-readable profile must be refused: %v", err)
	}
	for _, bad := range []string{"", "../x", "A", "a b", "a/b", strings.Repeat("a", 65)} {
		if _, err := Path(bad); err == nil {
			t.Errorf("accepted name %q", bad)
		}
	}
}

func TestResolve_environmentWinsAndMissingIsActionable(t *testing.T) {
	dir(t)
	Save("p", map[string]string{"secret": "from-profile", "plain": "host"})
	t.Setenv("P_SECRET", "from-env")
	fields := []Field{{Key: "secret", Label: "Secret", Hidden: true, Env: "P_SECRET"}, {Key: "plain", Label: "Plain"}}
	v, src, err := Resolve("p", fields)
	if err != nil || v["secret"] != "from-env" || src["secret"] != "environment" || v["plain"] != "host" || src["plain"] != "profile" {
		t.Errorf("resolve: %v %v %v", v, src, err)
	}

	fields = append(fields, Field{Key: "token", Label: "API token", Hidden: true, Env: "P_TOKEN"}, Field{Key: "extra", Optional: true})
	_, _, err = Resolve("p", fields)
	var nc *NotConfiguredError
	if !errors.As(err, &nc) || len(nc.Missing) != 1 || nc.Missing[0].Key != "token" {
		t.Fatalf("want one missing field: %v", err)
	}
	msg := err.Error()
	for _, want := range []string{"configure_tool", `"profile"`, `"key":"token"`, `"hidden":true`, "P_TOKEN"} {
		if !strings.Contains(msg+` "profile"`, want) {
			t.Errorf("the refusal lacks %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "from-profile") || strings.Contains(msg, "from-env") {
		t.Errorf("a value leaked into the error: %s", msg)
	}
}

func TestProtected_profilesAndSecretsFoldersAreNeverReadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvDir, filepath.Join(root, "profiles"))
	t.Setenv(EnvSecretsDir, filepath.Join(root, "secrets"))
	Save("p", map[string]string{"a": "1"})
	os.MkdirAll(filepath.Join(root, "secrets"), 0o700)
	os.WriteFile(filepath.Join(root, "secrets", "x.secret"), []byte("s"), 0o600)
	os.MkdirAll(filepath.Join(root, "work"), 0o700)

	for _, p := range []string{
		filepath.Join(root, "profiles"), filepath.Join(root, "profiles", "p.json"), filepath.Join(root, "profiles", "new.json"),
		filepath.Join(root, "profiles", "sub", "deeper.json"), filepath.Join(root, "secrets", "x.secret"), filepath.Join(root, "work", "..", "profiles", "p.json"),
	} {
		if !Protected(p) {
			t.Errorf("%s should be protected", p)
		}
	}
	for _, p := range []string{filepath.Join(root, "work", "a.txt"), filepath.Join(root, "profiles-not"), filepath.Join(root, "secretsx")} {
		if Protected(p) {
			t.Errorf("%s should not be protected", p)
		}
	}
	// A link in the work folder that points into the profiles folder is seen for what it is.
	link := filepath.Join(root, "work", "innocent.json")
	if err := os.Symlink(filepath.Join(root, "profiles", "p.json"), link); err != nil {
		t.Fatal(err)
	}
	if !Protected(link) {
		t.Error("a symlink into the profiles folder must be protected")
	}
}
