// Package profiles holds named credential profiles: sets of key/value pairs a tool reads at call
// time, kept out of the chat, the tool arguments, and every workspace.
//
// A profile is one JSON file, <name>.json, in a directory only its owner can use (0700), readable
// only by its owner (0600). A tool asks for the fields it needs; each is taken from an environment
// variable if one is set (the environment always wins), else from the profile. A field that is
// found in neither is reported as a NotConfiguredError, whose text tells the model exactly what to
// ask the user to configure, in a form it can copy into a configure_tool call. The values are
// never part of any error or result.
//
// File tools must never hand these files to a model: Protected reports whether a path is in the profiles directory or the
// issued-secrets directory, and mcp-local's file and IDE tools refuse such paths whatever folders are open.
//
// Who writes a profile: the user, in an editor, or through the Synod form (the extension host
// writes the file; the model never sees the values), or a provisioning tool that has just created
// a credential. Save merges into the existing file so one writer does not erase another's keys.
//
// # Goals
//
//   - One place a tool finds a credential that the user, not the model, supplied.
//   - Refuse a profile that others can read, and say how to fix it.
//   - Make the "not configured" refusal actionable by an agent without exposing a value.
//
// # Non-goals
//
//   - Encryption at rest. The file is protected by its permissions, like the issued secrets next
//     to it; a keychain store would sit behind the same Load and Save.
//   - Prompting. This package never asks anyone for anything.
//
// # Boundaries
//
// Owns: the profile file format, its location and permissions, and field resolution.
// Depends on: the standard library only.
// Used by: mcp-local (ledger_copy target), mcp-admin (provisioning), the Synod extension (which
// writes the same files from TypeScript).
package profiles
