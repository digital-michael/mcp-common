// Package bootstrap is the shared entry point of an MCP server binary built on a registry.Registry.
//
// Run parses the command line and either serves MCP over stdio (the default) or answers one of
// the introspection and direct-call modes: -version, -list-tools, -help and -run. Domain
// enablement is read from the environment through package config, and the registry is registered
// with the server under the binary's own family, scope and tier allow-list.
//
// # Goals
//
//   - Every server binary gets the same command line and server start-up without copying it.
//   - Direct invocation (-run) stays available for testing and scripting a tool without a client.
//
// # Non-goals
//
//   - Deciding which tiers a scope permits. The consuming binary owns that mapping and passes
//     the allow-list in.
//   - Any transport other than stdio.
//   - Tool implementations or tool metadata. Those are the registry's and the consumer's.
//
// # Boundaries
//
// Owns: the flag set, the stdio server start-up and the text of -list-tools and -help.
// Depends on: registry, config and the mcp-go library.
// Used by: mcp-admin, mcp-agent and the example server in this module.
package bootstrap
