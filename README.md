# mcp-common

Shared registry framework for the `mcp-*` family of MCP servers (currently
[`mcp-local`](https://github.com/digital-michael/mcp-local); a private `mcp-admin` is
planned). Provides one generic tool registry instead of the hand-maintained,
independently-drifting `Describe()`/`handlers`/`RegisterAll()` lists that pattern requires per
binary — see each package's doc comment for details.

## Packages

- `registry` — `Tool`/`Meta`/`Tier` types and the `Registry` that derives tool listing, CLI
  dispatch, and MCP server registration from one set of registered `ToolDef`s, stamping
  `_meta.family`/`_meta.scope` automatically on every tool.
- `config` — generic `MCP_TOOLS_<GROUP>` env-var enablement loading, keyed by domain name
  rather than a hardcoded struct field per domain.
- `bootstrap` — a reusable CLI + MCP-stdio-server entry point (`-list-tools`, `-run`, `-help`,
  default server mode) parameterized by a `*registry.Registry`.

## Design notes

- `family`/`scope` are caller-supplied strings, not hardcoded constants — this library doesn't
  know what "cortex" is. A consuming binary's own build-tag-gated files (e.g. `//go:build
  federated`) define its scope; mcp-common only knows how to stamp whatever string it's given.
  See `examples/echo-server` for the pattern.
- MCP's `_meta` field is a programmatic/client-side channel, not something a calling model
  sees — verified directly against a real MCP client before this library was designed around
  it. Use it for tool-set filtering upstream of an agent, not for model-facing preference.

## Status

Early scaffold. No consumers yet — `mcp-local` migration is separate, deliberately incremental
follow-up work.
