// Package config reads which tool domains are switched on.
//
// LoadEnabled returns an on or off flag for each domain name given, read from the environment
// variable MCP_TOOLS_<DOMAIN> in upper case. An unset or malformed value means enabled, so a
// forgotten variable never silently removes a domain.
//
// # Goals
//
//   - Domain switches driven by the names a binary's registry actually has, with no code change
//     when a domain is added.
//
// # Non-goals
//
//   - Access control. Enablement is operational (what this process offers), not a security
//     boundary; tiers are the boundary.
//   - Configuration beyond domain switches.
//
// # Boundaries
//
// Owns: the MCP_TOOLS_<DOMAIN> convention and its default.
// Depends on: the standard library only.
// Used by: bootstrap.
package config
