// Command echo-server is a minimal real consumer of this module: a one-tool MCP server built on
// registry, config and bootstrap, which proves they build and run end to end.
//
// It also shows the intended scope pattern. A consuming binary declares its own BuildScope
// string and AllowedTiers list (here in the scope files, one per build tag); this module never
// knows what "local" or "membership" mean, it only stamps the strings it is given.
//
// # Goals
//
//   - A runnable reference for how a binary uses the library, kept small enough to read in one sitting.
//   - A live check that version information set with linker flags crosses the module boundary.
//
// # Non-goals
//
//   - A product or a deployable server. It is an example and a test fixture.
//   - Showing every library feature.
//
// # Boundaries
//
// Owns: the example's scope files and its one echo tool.
// Depends on: bootstrap, buildinfo and registry.
// Used by: nobody; it is built by this module's Makefile.
//
// # Intentional failure
//
// INTENTIONAL-FAILURE: this command fails to compile with no scope tag (undefined: BuildScope).
// Exactly one of local, membership or global must be set, and there is no untagged scope file on
// purpose. Do not add a default to make a bare go build pass. Verify with: make test.
package main
