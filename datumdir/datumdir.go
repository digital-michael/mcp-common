// Package datumdir says where Datum Core keeps a machine's own files: one rule for the whole platform, so a path is not
// worked out again (and slightly differently) in every server and script.
//
// There are three folders, one per kind of file, all under the user's home:
//
//	config  what this machine or this user chose: model routes, personal aliases, agent definitions, the gateway's configuration folder
//	data    what the platform stores and the user would miss: the conversation history, credential profiles, issued secrets, vector data
//	state   what can be recreated: logs, job folders, the published workspaces
//
// Resolution, first match wins, for each kind:
//
//  1. DATUM_CONFIG_DIR / DATUM_DATA_DIR / DATUM_STATE_DIR, the folder itself
//  2. macOS and Linux: $XDG_CONFIG_HOME/datum, $XDG_DATA_HOME/datum, $XDG_STATE_HOME/datum
//  3. macOS and Linux: ~/.config/datum, ~/.local/share/datum, ~/.local/state/datum
//     Windows: %APPDATA%\datum, %LOCALAPPDATA%\datum, %LOCALAPPDATA%\datum\state
//
// macOS follows the XDG layout, as the platform did before this package existed (not ~/Library/Application Support), so nothing
// moves. Windows is implemented from the platform's documented folders and is NOT verified on a Windows machine: it is tested
// here only by giving the rule a fake environment.
//
// A setting that is more specific still wins over all of this: DATUM_MODELS_FILE, DATUM_PROFILES_DIR, DATUM_ADMIN_SECRETS_DIR and the
// like name one file or folder directly, and each server checks its own before it asks this package.
//
// The product's defaults ship in the repositories; nothing here is read from a repository (see docs/configuration.md).
//
// Used by: mcp-local, mcp-agent, mcp-admin and the servers that follow. The extension has the same rule in src/ids.ts, and a test on
// each side keeps them equal.
package datumdir

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Kind is one of the three folders.
type Kind string

const (
	Config Kind = "config"
	Data   Kind = "data"
	State  Kind = "state"
)

// Names of the variables that set a folder directly.
const (
	EnvConfig = "DATUM_CONFIG_DIR"
	EnvData   = "DATUM_DATA_DIR"
	EnvState  = "DATUM_STATE_DIR"
)

// Env is the world the rule reads, so a test can describe any machine.
type Env struct {
	GOOS   string              // "darwin", "linux", "windows"
	Home   string              // the user's home folder
	Getenv func(string) string // environment lookup
}

// Current is this process's environment. A missing home folder is reported by Dir, not here.
func Current() Env {
	home, _ := os.UserHomeDir()
	return Env{GOOS: runtime.GOOS, Home: home, Getenv: os.Getenv}
}

var errNoHome = errors.New("datumdir: no home folder: set DATUM_CONFIG_DIR, DATUM_DATA_DIR or DATUM_STATE_DIR")

// Dir is the folder for a kind of file in this environment. It does not create it.
func (e Env) Dir(k Kind) (string, error) {
	get := e.Getenv
	if get == nil {
		get = func(string) string { return "" }
	}
	var direct, xdg, sub string
	switch k {
	case Config:
		direct, xdg, sub = EnvConfig, "XDG_CONFIG_HOME", ".config"
	case Data:
		direct, xdg, sub = EnvData, "XDG_DATA_HOME", filepath.Join(".local", "share")
	case State:
		direct, xdg, sub = EnvState, "XDG_STATE_HOME", filepath.Join(".local", "state")
	default:
		return "", errors.New("datumdir: unknown kind " + string(k))
	}
	if d := get(direct); d != "" {
		return d, nil
	}
	if e.GOOS == "windows" {
		return e.windows(k, get)
	}
	if x := get(xdg); x != "" {
		return filepath.Join(x, "datum"), nil
	}
	if e.Home == "" {
		return "", errNoHome
	}
	return filepath.Join(e.Home, sub, "datum"), nil
}

// windows applies the documented folders: %APPDATA% for configuration, %LOCALAPPDATA% for what stays on the machine.
func (e Env) windows(k Kind, get func(string) string) (string, error) {
	// The path package of the host would join with the host's separator; a Windows rule is written with Windows ones.
	join := func(parts ...string) string {
		out := parts[0]
		for _, p := range parts[1:] {
			out += `\` + p
		}
		return out
	}
	app, local := get("APPDATA"), get("LOCALAPPDATA")
	if app == "" && e.Home != "" {
		app = join(e.Home, "AppData", "Roaming")
	}
	if local == "" && e.Home != "" {
		local = join(e.Home, "AppData", "Local")
	}
	switch k {
	case Config:
		if app != "" {
			return join(app, "datum"), nil
		}
	case Data:
		if local != "" {
			return join(local, "datum"), nil
		}
	case State:
		if local != "" {
			return join(local, "datum", "state"), nil
		}
	}
	return "", errNoHome
}

// ConfigDir, DataDir and StateDir are Dir for this process's environment.
func ConfigDir() (string, error) { return Current().Dir(Config) }
func DataDir() (string, error)   { return Current().Dir(Data) }
func StateDir() (string, error)  { return Current().Dir(State) }
