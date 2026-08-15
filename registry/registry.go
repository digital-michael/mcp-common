// Package registry provides one generic tool registry for an MCP server,
// replacing the pattern of hand-maintaining three separate, independently
// drifting lists for the same tool set (a Describe()-style metadata slice, a
// name->handler dispatch map, and a per-domain server-registration
// call-list) -- confirmed as mcp-local's actual current shape before this
// package was designed. A consumer registers each domain's tools once, as a
// []ToolDef; tool listing, direct CLI dispatch, and MCP server registration
// are all derived from that one registration, not re-authored separately.
package registry

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Tier classifies who/where a tool may be called from. TierInternal is the
// zero value deliberately -- an unset Tier is treated as Internal (never
// registered with the MCP server, reachable only via direct Run), so a
// forgotten classification makes a tool invisible rather than accidentally
// exposed.
type Tier string

const (
	TierInternal  Tier = "Internal"
	TierLocal     Tier = "Local"
	TierFederated Tier = "Federated"
	TierGlobal    Tier = "Global"
	TierAdmin     Tier = "Admin"
)

// HandlerFunc is the standard mcp-go tool handler signature.
type HandlerFunc func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// CLIInfo carries the human-readable rendering hints a bootstrap CLI needs
// for -list-tools/-help/-run (positional-argument mapping, example values).
// It is deliberately separate from mcp.Tool's real JSON input schema, which
// remains the source of truth for what an MCP client actually sends.
type CLIInfo struct {
	Args      string            // human-readable one-line argument summary
	Params    []string          // ordered param names for -run positional mapping (required first)
	RequiredN int               // number of leading Params that are required
	Examples  map[string]string // example values keyed by param name, for -help
}

// ToolDef is one tool's complete registration: its real mcp.Tool (name,
// description, and full input schema, built the same way as today via
// mcp.NewTool(name, opts...)), its handler, which domain it belongs to, its
// access tier, and its CLI rendering hints.
type ToolDef struct {
	Tool    mcp.Tool
	Handler HandlerFunc
	Domain  string
	Tier    Tier
	CLI     CLIInfo
}

// Registry holds every registered ToolDef in registration order and an
// index by name for O(1) lookup (mcp-local's current FindMeta is an O(n)
// linear scan that reallocates its whole source slice on every call).
type Registry struct {
	defs   []ToolDef
	byName map[string]int
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{byName: make(map[string]int)}
}

// Add registers one or more ToolDefs. Panics on a duplicate tool name or an
// empty name -- both are programming errors that should fail at startup,
// not surface as a confusing runtime lookup miss later.
func (r *Registry) Add(defs ...ToolDef) {
	for _, def := range defs {
		name := def.Tool.Name
		if name == "" {
			panic("registry: ToolDef registered with empty tool name")
		}
		if _, exists := r.byName[name]; exists {
			panic(fmt.Sprintf("registry: duplicate tool name %q", name))
		}
		r.byName[name] = len(r.defs)
		r.defs = append(r.defs, def)
	}
}

// Describe returns every registered tool's mcp.Tool plus CLI/Domain/Tier
// info, in registration order. Unlike mcp-local's Describe(), this always
// includes every tool regardless of tier or enablement -- callers filter as
// needed; Describe itself never hides anything, since -help/-run should
// work even for a disabled or non-Global tool run directly.
func (r *Registry) Describe() []ToolDef {
	out := make([]ToolDef, len(r.defs))
	copy(out, r.defs)
	return out
}

// FindMeta returns the ToolDef registered under name, if any.
func (r *Registry) FindMeta(name string) (ToolDef, bool) {
	i, ok := r.byName[name]
	if !ok {
		return ToolDef{}, false
	}
	return r.defs[i], true
}

// Run invokes name's handler directly, bypassing the stdio/JSON-RPC
// transport -- the mechanism behind a bootstrap CLI's -run flag. Tier and
// enablement are NOT checked here; Run is a direct, trusted, local
// invocation path (matching mcp-local's existing Run semantics), not a
// remote-request path.
func (r *Registry) Run(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	def, ok := r.FindMeta(name)
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return def.Handler(ctx, req)
}

// RegisterWithServer registers every enabled tool with s, stamping
// _meta.family and _meta.scope on each one automatically -- no per-tool or
// per-domain code needs to remember to do this. A tool is registered only
// if enabled[def.Domain] is true; a domain absent from enabled is treated
// as enabled by default (fail-open on enablement, matching mcp-local's
// existing MCP_TOOLS_<GROUP> behavior -- this is a deliberate match, not an
// independent choice; see config.LoadEnabled's doc comment for the
// reasoning). Tier is otherwise NOT filtered here -- which tiers a given
// binary should expose beyond Internal is a build-time/deployment decision
// the caller makes by choosing what it Adds to the Registry in the first
// place, not something this method decides at registration time. The one
// exception is TierInternal itself: it is always skipped, unconditionally,
// matching Tier's own doc comment ("never registered with the MCP server,
// reachable only via direct Run") -- Run() stays tier-blind by design, so
// this is the only enforcement point for that guarantee.
func (r *Registry) RegisterWithServer(s *server.MCPServer, family, scope string, enabled map[string]bool) {
	for _, def := range r.defs {
		if def.Tier == TierInternal {
			continue
		}
		if on, explicit := enabled[def.Domain]; explicit && !on {
			continue
		}
		t := def.Tool
		if t.Meta == nil {
			t.Meta = &mcp.Meta{}
		}
		if t.Meta.AdditionalFields == nil {
			t.Meta.AdditionalFields = make(map[string]any)
		}
		t.Meta.AdditionalFields["family"] = family
		t.Meta.AdditionalFields["scope"] = scope
		s.AddTool(t, server.ToolHandlerFunc(def.Handler))
	}
}
