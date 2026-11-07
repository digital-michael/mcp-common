// Package registry is the single place an MCP server registers its tools.
//
// A consumer registers each domain's tools once, as a []ToolDef: the tool itself (name,
// description, input schema), its handler, its domain, its Tier (who may call it), its Reach
// (what a call touches) and its command-line hints. Tool listing, direct command-line dispatch
// and MCP server registration are all derived from that one registration, so the three lists a
// server used to maintain by hand cannot drift apart.
//
// Registration is filtered, not trusted. RegisterWithServer registers a tool only if its Tier is
// in the allow-list the consuming binary passes in and its domain is enabled, and stamps
// _meta.family, _meta.scope and _meta.domain on every tool it registers. Internal tools are never
// registered with the server; a tool with no Tier is in no allow-list and is not registered
// either. Run, the direct call path, is deliberately tier-blind: it is a trusted local call, not
// a remote request. Access (Read, Write or Publish, with Destructive and OpenWorld) is the single
// source of a tool's MCP annotations and description prefix, and Classify refuses to proceed while
// any registered tool is unclassified.
//
// # Goals
//
//   - One registration per tool, from which listing, dispatch and server registration derive.
//   - Fail closed: an unclassified tool or an unset tier makes a tool invisible, never exposed.
//   - Keep Tier (who may call a tool) and Reach (what a call touches) as separate questions.
//   - Serve the platform goal of giving models real tools, safely.
//
// # Non-goals
//
//   - Authentication of callers. Tier filtering governs what a binary can expose, not who may
//     connect to it; membership authentication does not exist yet.
//   - Knowing any product's names. Family and scope are strings the consumer supplies.
//   - Tool implementations. Those live in the consuming binary.
//
// # Boundaries
//
// Owns: ToolDef, Tier, Reach, Access and Class, the Registry, tier and enablement filtering, and
// the _meta stamp.
// Depends on: the mcp-go library only.
// Used by: bootstrap and buildinfo in this module, and by mcp-local, mcp-admin and mcp-agent.
package registry
