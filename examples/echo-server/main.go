// Command echo-server is a minimal real consumer of mcp-common, proving the
// registry/config/bootstrap packages build and run end to end. It also
// demonstrates the intended build-tag pattern: BuildScope (defined in
// scope_local.go/scope_membership.go/scope_global.go, one active per build)
// lives here, in the consuming binary -- mcp-common itself never knows what
// "cortex" or "local"/"membership"/"global" mean, it only stamps whatever
// family/scope strings it's given.
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/bootstrap"
	"github.com/digital-michael/mcp-common/buildinfo"
	"github.com/digital-michael/mcp-common/registry"
)

// family is this binary's own identity, not mcp-common's -- a real consumer
// (mcp-local, mcp-admin) would set this to whatever product name applies.
const family = "cortex-example"

func echoHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, _ := req.GetArguments()["message"].(string)
	return mcp.NewToolResultText(message), nil
}

func main() {
	reg := registry.New()
	reg.Add(registry.ToolDef{
		Tool: mcp.NewTool("echo",
			mcp.WithDescription("Echoes a message back."),
			mcp.WithString("message", mcp.Required(), mcp.Description("Message to echo.")),
		),
		Handler: echoHandler,
		Domain:  "diagnostics",
		Tier:    registry.TierGlobal,
		CLI: registry.CLIInfo{
			Args:      "message: string (required)",
			Params:    []string{"message"},
			RequiredN: 1,
			Examples:  map[string]string{"message": "Hello, mcp-common!"},
		},
	})
	// Proves the ldflags-across-package-boundary mechanism works before
	// mcp-admin ever depends on it -- see the Makefile's build target for
	// the actual -X flags that stamp buildinfo.Version/BuildID.
	reg.Add(buildinfo.Tool("echo-server", registry.TierGlobal))

	bootstrap.Run("echo-server", buildinfo.Full(), family, BuildScope, reg, []string{"diagnostics"})
}
