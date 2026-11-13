package admintoken

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func use(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "secrets", FileName)
	t.Setenv(EnvFile, p)
	return p
}

func TestGateIsClosedWithoutAToken(t *testing.T) {
	use(t)
	if err := Check("anything"); !errors.Is(err, ErrNotSet) {
		t.Fatalf("no file must refuse everything, got %v", err)
	}
	if err := Check(""); !errors.Is(err, ErrNotSet) {
		t.Fatalf("empty with no file: %v", err)
	}
}

func TestCreateCheckAndReplace(t *testing.T) {
	p := use(t)
	got, err := Create(false)
	if err != nil || got != p {
		t.Fatalf("create: %q %v", got, err)
	}
	st, _ := os.Stat(p)
	dir, _ := os.Stat(filepath.Dir(p))
	if st.Mode().Perm() != 0o600 || dir.Mode().Perm() != 0o700 {
		t.Errorf("modes: file %v folder %v", st.Mode().Perm(), dir.Mode().Perm())
	}
	tok, err := Load()
	if err != nil || len(tok) != 64 {
		t.Fatalf("load: %q %v", tok, err)
	}
	if err := Check(tok); err != nil {
		t.Errorf("right token: %v", err)
	}
	if err := Check(tok + "\n"); err != nil {
		t.Errorf("surrounding whitespace is ignored: %v", err)
	}
	for _, bad := range []string{"", "nope", tok[:63], tok + "x"} {
		if Check(bad) == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
	if _, err := Create(false); err == nil {
		t.Error("an existing token is not replaced by accident")
	}
	if _, err := Create(true); err != nil {
		t.Fatal(err)
	}
	if Check(tok) == nil {
		t.Error("the old token stops working once replaced (read at every call, not cached)")
	}
}

func TestShortOrEmptyFileIsNeverAnOpenGate(t *testing.T) {
	p := use(t)
	os.MkdirAll(filepath.Dir(p), 0o700)
	for _, body := range []string{"", "\n", "short"} {
		os.WriteFile(p, []byte(body), 0o600)
		if err := Check(body); err == nil || errors.Is(err, ErrNotSet) {
			t.Errorf("%q: want an invalid-token error, got %v", body, err)
		}
	}
}

func TestDefaultPathIsInsideAProtectedFolder(t *testing.T) {
	t.Setenv(EnvFile, "")
	t.Setenv("DATUM_ADMIN_SECRETS_DIR", filepath.Join(t.TempDir(), "secrets"))
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != FileName || filepath.Base(filepath.Dir(p)) != "secrets" {
		t.Errorf("path: %s", p)
	}
}
