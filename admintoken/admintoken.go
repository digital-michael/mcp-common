package admintoken

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/digital-michael/mcp-common/profiles"
	"github.com/digital-michael/mcp-common/registry"
)

// EnvFile overrides where the token is kept.
const EnvFile = "DATUM_ADMIN_TOKEN_FILE"

// FileName is the token's file name inside the protected secrets folder.
const FileName = "model-admin-token"

// ArgName is the tool argument that carries the token.
const ArgName = "admin_token"

// ErrNotSet means no token file exists: the gate is closed until one is created.
var ErrNotSet = errors.New("the admin token has not been created")

// Path is the token file: DATUM_ADMIN_TOKEN_FILE, else FileName in the first protected secrets folder.
func Path() (string, error) {
	if p := os.Getenv(EnvFile); p != "" {
		return p, nil
	}
	dirs := profiles.ProtectedDirs()
	if len(dirs) < 2 {
		return "", errors.New("cannot find the secrets folder")
	}
	return filepath.Join(dirs[1], FileName), nil
}

// Load reads the token. A missing file is ErrNotSet; an empty or short one is an error, never an open gate.
func Load() (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return "", ErrNotSet
	}
	if err != nil {
		return "", err
	}
	t := strings.TrimSpace(string(b))
	if len(t) < 32 {
		return "", fmt.Errorf("%s holds an invalid token (it must be at least 32 characters); replace it", p)
	}
	return t, nil
}

// Create writes a new random token (private file, private folder) and returns its path. It refuses to replace an existing
// one unless replace is true.
func Create(replace bool) (string, error) {
	p, err := Path()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil && !replace {
		return p, fmt.Errorf("a token already exists at %s", p)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(hex.EncodeToString(raw)+"\n"), 0o600); err != nil {
		return "", err
	}
	return p, os.Rename(tmp, p)
}

// Check compares a given token with the stored one in constant time. It returns nil only for a match.
func Check(given string) error {
	want, err := Load()
	if err != nil {
		return err
	}
	if given == "" {
		return errors.New("this tool needs the admin token (admin_token)")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(given)), []byte(want)) != 1 {
		return errors.New("the admin token is wrong")
	}
	return nil
}

// Gate adds the admin_token argument to a tool and refuses the call, before the tool's own handler runs, unless it matches the
// stored token. A refused call changes nothing, not even by a dry run.
func Gate(d registry.ToolDef) registry.ToolDef {
	// The schema's map is shared with the definition passed in: copy it, so wrapping never changes the original.
	props := make(map[string]any, len(d.Tool.InputSchema.Properties)+1)
	for k, v := range d.Tool.InputSchema.Properties {
		props[k] = v
	}
	d.Tool.InputSchema.Properties = props
	d.Tool.InputSchema.Properties[ArgName] = map[string]any{
		"type":        "string",
		"description": "The admin token. Required: this tool changes things, and a gateway key alone is not enough. The chat supplies it; it is kept in the user's protected secrets folder.",
	}
	d.CLI.Params = append(append([]string(nil), d.CLI.Params...), ArgName)
	d.CLI.Args += ", " + ArgName + " (required)"
	ex := map[string]string{ArgName: "<token>"}
	for k, v := range d.CLI.Examples {
		ex[k] = v
	}
	d.CLI.Examples = ex
	inner := d.Handler
	d.Handler = func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := Check(req.GetString(ArgName, "")); err != nil {
			msg := err.Error()
			if errors.Is(err, ErrNotSet) {
				msg = "no admin token exists yet. In the chat the first change that needs it offers to create it"
			}
			return mcp.NewToolResultError("NOT ALLOWED, nothing was changed: " + msg), nil
		}
		return inner(ctx, req)
	}
	return d
}
