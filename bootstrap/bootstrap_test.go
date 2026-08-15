package bootstrap

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/registry"
)

func testHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.GetArguments()["name"].(string)
	return mcp.NewToolResultText("hello " + name), nil
}

func testRegistry() *registry.Registry {
	r := registry.New()
	r.Add(registry.ToolDef{
		Tool:    mcp.NewTool("greet", mcp.WithDescription("test tool")),
		Handler: testHandler,
		Domain:  "diagnostics",
		Tier:    registry.TierGlobal,
		CLI: registry.CLIInfo{
			Args:      "name: string (required)",
			Params:    []string{"name"},
			RequiredN: 1,
			Examples:  map[string]string{"name": "world"},
		},
	})
	return r
}

func TestLooksLikeKeyValue(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"world"}, false},
		{[]string{"name=world"}, true},
		{[]string{"name=world", "extra=1"}, true},
		{[]string{"name=world", "bare"}, false},
	}
	for _, c := range cases {
		if got := looksLikeKeyValue(c.args); got != c.want {
			t.Errorf("looksLikeKeyValue(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestRunToolCmdPositional(t *testing.T) {
	reg := testRegistry()
	if err := runToolCmd(reg, "greet", []string{"world"}); err != nil {
		t.Fatalf("runToolCmd positional error = %v", err)
	}
}

func TestRunToolCmdKeyValue(t *testing.T) {
	reg := testRegistry()
	if err := runToolCmd(reg, "greet", []string{"name=world"}); err != nil {
		t.Fatalf("runToolCmd key=value error = %v", err)
	}
}

func TestRunToolCmdMissingRequired(t *testing.T) {
	reg := testRegistry()
	if err := runToolCmd(reg, "greet", nil); err == nil {
		t.Fatal("runToolCmd with missing required arg: error = nil, want non-nil")
	}
}

func TestRunToolCmdUnknownTool(t *testing.T) {
	reg := testRegistry()
	if err := runToolCmd(reg, "no-such-tool", nil); err == nil {
		t.Fatal("runToolCmd with unknown tool: error = nil, want non-nil")
	}
}

func TestRunToolCmdTooManyPositional(t *testing.T) {
	reg := testRegistry()
	if err := runToolCmd(reg, "greet", []string{"world", "extra"}); err == nil {
		t.Fatal("runToolCmd with too many positional args: error = nil, want non-nil")
	}
}
