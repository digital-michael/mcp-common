// Package buildinfo is the shared version and build identity of a binary.
//
// Version and BuildID are set at build time with linker flags, for example:
//
//	go build -ldflags "-X github.com/digital-michael/mcp-common/buildinfo.Version=1.2.3 -X github.com/digital-michael/mcp-common/buildinfo.BuildID=20260815120000" ...
//
// A plain build with no flags reports "0.1.0+dev". Tool returns a ready-to-register "version" tool
// for a binary, at the tier the caller chooses.
//
// # Goals
//
//   - One version convention for every binary in the family, set the same way.
//   - A version tool that reports the binary actually running, so a stale install is visible.
//
// # Non-goals
//
//   - Deciding who may call the version tool. The caller supplies its tier.
//   - Release management or version bumping.
//
// # Boundaries
//
// Owns: the Version and BuildID variables, Full and the version tool constructor.
// Depends on: registry and the mcp-go library.
// Used by: mcp-admin, mcp-agent and the example server in this module.
package buildinfo
