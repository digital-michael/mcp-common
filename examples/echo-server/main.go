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
const family = "datum-example"

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
		Tier:    registry.TierLocal,
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
	reg.Add(buildinfo.Tool("echo-server", registry.TierLocal))

	bootstrap.Run("echo-server", buildinfo.Full(), family, BuildScope, AllowedTiers, reg, []string{"diagnostics"})
}
