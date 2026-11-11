// Package modelprefs holds one user's personal model settings: aliases (a label of the user's own for a
// model) and the context size the user wants for a model.
//
// The settings are one JSON file on the machine the user works on, ~/.config/datum/models.json
// (DATUM_MODELS_FILE moves it). They are deliberately not in the inference gateway's configuration:
// on a shared deployment that configuration belongs to every user, and a personal alias or context
// size must not change it. The gateway therefore never sees an alias. The client resolves it to the
// real model name before it sends a request, and sends the user's context size (num_ctx) with the
// request, which overrides the gateway's default for that request alone.
//
// # Goals
//
//   - One file, one reader, shared by every program that names a model for the user.
//   - Aliases that cannot hide or shadow a real model, and cannot chain.
//
// # Non-goals
//
//   - Secrets. The file holds no credentials (see package profiles for those).
//   - Knowing which models exist. The caller says whether a name is already a real model.
//
// # Boundaries
//
// Owns: the file format, its location, and the alias rules.
// Depends on: the standard library only.
// Used by: mcp-local (model tools), and the Synod extension, which reads and writes the same file.
package modelprefs
