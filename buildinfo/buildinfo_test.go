package buildinfo

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/registry"
)

func TestFullDefaults(t *testing.T) {
	if got := Full(); got != "0.1.0+dev" {
		t.Fatalf("Full() = %q, want %q (source defaults, unless ldflags override)", got, "0.1.0+dev")
	}
}

func TestFullReflectsOverride(t *testing.T) {
	origVersion, origBuildID := Version, BuildID
	defer func() { Version, BuildID = origVersion, origBuildID }()

	Version = "1.2.3"
	BuildID = "20260815120000"
	if got, want := Full(), "1.2.3+20260815120000"; got != want {
		t.Fatalf("Full() = %q, want %q", got, want)
	}
}

func TestToolShape(t *testing.T) {
	def := Tool("test-binary", registry.TierAdmin)

	if def.Tool.Name != "version" {
		t.Errorf("Tool.Name = %q, want %q", def.Tool.Name, "version")
	}
	if def.Domain != "diagnostics" {
		t.Errorf("Domain = %q, want %q", def.Domain, "diagnostics")
	}
	if def.Tier != registry.TierAdmin {
		t.Errorf("Tier = %q, want %q", def.Tier, registry.TierAdmin)
	}
}

func TestToolHandlerReportsBinaryName(t *testing.T) {
	origVersion, origBuildID := Version, BuildID
	defer func() { Version, BuildID = origVersion, origBuildID }()
	Version, BuildID = "9.9.9", "test-build"

	def := Tool("my-binary", registry.TierGlobal)
	result, err := def.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Handler error = %v", err)
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("Handler result has no text content: %v", result)
	}
	if want := "my-binary 9.9.9+test-build"; text.Text != want {
		t.Fatalf("Handler text = %q, want %q", text.Text, want)
	}
}

func TestToolDifferentTiersIndependent(t *testing.T) {
	admin := Tool("bin-a", registry.TierAdmin)
	global := Tool("bin-b", registry.TierGlobal)

	if admin.Tier != registry.TierAdmin {
		t.Errorf("admin.Tier = %q, want Admin", admin.Tier)
	}
	if global.Tier != registry.TierGlobal {
		t.Errorf("global.Tier = %q, want Global", global.Tier)
	}
}
