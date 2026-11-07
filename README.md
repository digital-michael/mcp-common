# mcp-common

The shared registry framework for the `mcp-*` family of MCP servers: one place to register a tool, from which listing, command-line dispatch and server registration all derive.

## Purpose

Every MCP server in the family (`mcp-local`, `mcp-admin`, `mcp-agent`) needs the same machinery: a registry of tools, a decision about which of them a given binary may expose, a command line for running a tool without a client, and a version tool. This module is that machinery, so each server writes only its own tools.

## Goals

- **One registration per tool.** Register a tool once; tool listing, direct dispatch and MCP registration are derived from it, so they cannot drift apart.
- **Fail closed.** An unclassified tool or an unset tier makes a tool invisible, never exposed.
- **Separate questions kept separate.** *Tier* says who may call a tool, *Reach* says what a call touches, and *domain* says which group the tool belongs to. Each is stamped on the tool, with the binary's family and scope.
- **Same behaviour in every server.** One command line, one version convention, one set of domain switches.
- Serve the platform goal of giving models real tools, safely.

## Non-goals

- **Authentication.** Tier filtering decides what a binary can expose, not who may connect to it. Authenticating callers of the membership tier does not exist yet.
- **Tool implementations.** They live in the consuming binary.
- **Product vocabulary.** Family and scope are strings the consumer supplies; this module does not know what any of them mean.
- **Transports other than stdio.**

## Boundaries

| | |
|---|---|
| **Owns** | `ToolDef`, `Tier`, `Reach`, `Access` and `Class`, the registry and its filtering, the `_meta` stamp, the server command line, domain switches, the version convention, the model-slot rule |
| **Depends on** | the [mcp-go](https://github.com/mark3labs/mcp-go) library; the standard library |
| **Used by** | `mcp-local`, `mcp-admin` and `mcp-agent` (all three import `registry`; `mcp-admin` and `mcp-agent` also use `bootstrap` and `buildinfo`; `mcp-agent` uses `slots`) |

The dependency direction is one way: consumers import this module, and this module imports none of them.

## Layout

| Package | What it is |
|---|---|
| `registry` | `ToolDef`, tiers, reach, access classification and the registry that registers, filters and stamps tools |
| `bootstrap` | the shared command line and stdio server start-up (`-version`, `-list-tools`, `-help`, `-run`) |
| `config` | domain switches from `MCP_TOOLS_<DOMAIN>` |
| `buildinfo` | version and build id set by linker flags, and a ready-made `version` tool |
| `slots` | how many model requests are served at once, from `DATUM_MODEL_SLOTS` and the machine's memory |
| `examples/echo-server` | a one-tool server showing the build-tag scope pattern |

Each package carries its own scope and goals in its `doc.go`; `go doc ./registry` shows it.

## Build and test

```sh
make build   # builds the echo-server example for each scope tag
make test    # vet and test the library packages, then the example under each tag
```

The library packages need no build tag. The example is a consumer and declares its scope with one.

## Related

- Licence: [LICENSE](LICENSE).
