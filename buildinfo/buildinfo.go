package buildinfo

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/registry"
)

var (
	Version = "0.1.0"
	BuildID = "dev"
)

// Full returns the combined version string, e.g. "1.2.3+20260815120000".
func Full() string {
	return fmt.Sprintf("%s+%s", Version, BuildID)
}

// Tool returns a ready-to-Add() "version" ToolDef reporting binaryName's
// build info. tier is caller-supplied -- different consumers may want this
// gated differently (a Local tool in one binary, Admin-only in another),
// so buildinfo itself stays tier-agnostic rather than assuming one answer.
func Tool(binaryName string, tier registry.Tier) registry.ToolDef {
	return registry.ToolDef{
		Tool: mcp.NewTool("version",
			mcp.WithDescription(fmt.Sprintf("Returns %s's version and build id.", binaryName)),
		),
		Handler: func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(fmt.Sprintf("%s %s", binaryName, Full())), nil
		},
		Domain: "diagnostics",
		Tier:   tier,
		CLI: registry.CLIInfo{
			Args: "(none)",
		},
	}
}
