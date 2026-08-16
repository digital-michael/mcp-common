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
	TierInternal Tier = "Internal"
	// TierLocal is the default/base tier: a tool callable from this machine.
	// Absorbed "Global" (2026-08-16) -- once Membership is understood as a
	// superset that trivially includes local access (you're a member of
	// your own network), a separate "safe everywhere" middle tier added no
	// real distinction: nothing was ever Membership-only, so every
	// previously-"Global" tool was already just as available as a Local
	// one. Local is now the default most tools get; Membership is reserved
	// for genuinely peer-only capability, not a promotion every tool needs.
	TierLocal Tier = "Local"
	// TierMembership identifies a tool only callable from a trusted peer
	// workstation within the fleet -- not this machine, not the open
	// internet/a third-party. Renamed from "Federated" (2026-08-16): that
	// name collided with unrelated meanings ("federated learning" in ML,
	// generic "any multi-node topology" in distributed systems) and caused
	// real confusion this session (conflated with a build tag, with
	// "Global," with inbound vs. outbound). "Membership" also matches
	// existing vocabulary already used elsewhere in this stack
	// (cortex-stack's litellm.yaml `cortex-membership-cloud-*` alias
	// ladder), so the whole family now uses one word for this concept.
	// Reserved -- no tool occupies this tier yet.
	TierMembership Tier = "Membership"
	TierAdmin      Tier = "Admin"
)

// Reach classifies what a tool's own execution touches, independent of Tier
// (who is allowed to call it). Added 2026-08-16 after a concrete case Tier
// alone couldn't express: a future remote-enabled `delegate` tool should
// stay Local/Global-callable (a purely local caller is its primary use
// case) while its own execution can reach a trusted peer's infrastructure
// -- tagging the TOOL itself Membership under the inbound-only Tier
// definition would make it unreachable from that primary caller. Tier
// answers "who can call this"; Reach answers "what does calling it touch."
type Reach string

const (
	// ReachLocal: stays on this machine -- filesystem, OS, sibling local
	// processes (e.g. spreadsheet-services). The default for nearly every
	// tool today; nothing currently reaches further.
	ReachLocal Reach = "Local"
	// ReachMembership: touches managed membership infrastructure -- shared
	// across trusted peer workstations (peer LiteLLM/model routing, a
	// shared DB/cache, team chat/workspaces). No tool uses this yet --
	// federated LiteLLM routing itself is still backlog (ledger a800512b).
	ReachMembership Reach = "Membership"
	// ReachCloud: touches a third-party, unmanaged external provider --
	// data leaves the membership boundary entirely. No tool uses this yet.
	ReachCloud Reach = "Cloud"
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
// access tier, its outbound reach, and its CLI rendering hints.
type ToolDef struct {
	Tool    mcp.Tool
	Handler HandlerFunc
	Domain  string
	Tier    Tier
	Reach   Reach
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
// work even for a disabled or restricted-tier tool run directly.
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

// RegisterWithServer registers every enabled, tier-allowed tool with s,
// stamping _meta.family and _meta.scope on each one automatically -- no
// per-tool or per-domain code needs to remember to do this.
//
// Two independent filters apply, in order:
//  1. Tier: a tool is registered only if its Tier appears in allowedTiers.
//     TierInternal is always skipped regardless of allowedTiers, matching
//     Tier's own doc comment ("never registered with the MCP server,
//     reachable only via direct Run") -- Run() stays tier-blind by design,
//     so this is the only enforcement point for that guarantee.
//  2. Enablement: a tool is registered only if enabled[def.Domain] is true;
//     a domain absent from enabled is treated as enabled by default
//     (fail-open, matching mcp-local's existing MCP_TOOLS_<GROUP> behavior
//     -- see config.LoadEnabled's doc comment for the reasoning).
//
// allowedTiers is caller-supplied rather than inferred from scope here,
// deliberately: this package doesn't own any one consumer's BuildScope
// vocabulary (mcp-local uses local/membership/all, mcp-admin uses
// admin/all, examples/echo-server uses local/membership/global -- three
// different tag sets), so "which tag permits which tiers" is a decision
// each consumer makes for itself (typically in its own scope_*.go files,
// alongside the BuildScope const already declared there) and passes in.
// This mirrors, and generalizes, the TierInternal precedent above: a
// runtime check at registration time, not a compile-time build-tag-gated
// registration call -- unlike mcp-admin's per-category register_X.go
// pattern, this generalizes to a domain containing tools of mixed tiers
// (e.g. a future `delegate` with one Local and one Membership tool).
func (r *Registry) RegisterWithServer(s *server.MCPServer, family, scope string, allowedTiers []Tier, enabled map[string]bool) {
	// TODO: membership authentication goes here. TierMembership currently
	// has no enforcement of caller identity (unlike Tier/allowedTiers
	// filtering just above, which only governs what a binary CAN expose,
	// not who's allowed to connect to it) -- a Membership-tier tool, once
	// one exists and is registered, is exposed to whatever caller reaches
	// this connection, with no verification that caller is actually a
	// trusted peer. litellm.yaml's mcp_servers config still has
	// allow_all_keys: true, and real per-caller authz (BL-008-012) is still
	// blocked. Deferred deliberately (operator, 2026-08-16) -- not being
	// addressed now, but the gap is real and this is where the check would
	// go once it exists. See docs/mcp-local/tool-access-tiers.md and ledger
	// epic 05fdfa65.
	for _, def := range r.filterForRegistration(allowedTiers, enabled) {
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

// filterForRegistration returns, in registration order, the subset of defs
// that RegisterWithServer would register -- the Tier/allowedTiers and
// Domain/enabled filtering logic, pulled out as a pure function so it's
// directly unit-testable. mcp-go's MCPServer exposes no lookup accessor in
// this version, so asserting "which tools got registered" against a real
// *server.MCPServer isn't possible from a test; this is what the actual
// filtering-correctness tests call instead. RegisterWithServer itself stays
// the only place that touches the server/meta-stamping side effects.
func (r *Registry) filterForRegistration(allowedTiers []Tier, enabled map[string]bool) []ToolDef {
	allowed := make(map[Tier]bool, len(allowedTiers))
	for _, t := range allowedTiers {
		allowed[t] = true
	}
	var out []ToolDef
	for _, def := range r.defs {
		if def.Tier == TierInternal {
			continue
		}
		if !allowed[def.Tier] {
			continue
		}
		if on, explicit := enabled[def.Domain]; explicit && !on {
			continue
		}
		out = append(out, def)
	}
	return out
}
