package admintoken

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/registry"
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

func TestGateWrapsAToolAndRefusesBeforeItRuns(t *testing.T) {
	use(t)
	ran := 0
	def := registry.ToolDef{
		Tool: mcp.NewTool("thing_write", mcp.WithDescription("x"), mcp.WithString("a")),
		Handler: func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ran++
			return mcp.NewToolResultText("ran"), nil
		},
		CLI: registry.CLIInfo{Args: "a: string", Params: []string{"a"}, Examples: map[string]string{"a": "1"}},
	}
	g := Gate(def)
	if _, ok := g.Tool.InputSchema.Properties[ArgName]; !ok || len(g.CLI.Params) != 2 || g.CLI.Examples[ArgName] == "" {
		t.Fatalf("the argument is declared: %+v %+v", g.Tool.InputSchema.Properties, g.CLI)
	}
	if _, leaked := def.Tool.InputSchema.Properties[ArgName]; leaked || len(def.CLI.Params) != 1 || def.CLI.Examples[ArgName] != "" {
		t.Error("the original definition is not changed (its schema map is shared, so it is copied)")
	}
	call := func(tok string) *mcp.CallToolResult {
		var req mcp.CallToolRequest
		req.Params.Arguments = map[string]any{ArgName: tok}
		r, _ := g.Handler(context.Background(), req)
		return r
	}
	if r := call("x"); !r.IsError || ran != 0 {
		t.Fatalf("no token file: refused, not run (%d)", ran)
	}
	if _, err := Create(false); err != nil {
		t.Fatal(err)
	}
	if r := call("wrong"); !r.IsError || ran != 0 {
		t.Fatal("wrong token refused")
	}
	tok, _ := Load()
	if r := call(tok); r.IsError || ran != 1 {
		t.Fatalf("right token runs the tool once (%d)", ran)
	}
}
