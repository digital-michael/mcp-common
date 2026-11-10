package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EnvDir overrides the profiles directory.
const EnvDir = "DATUM_PROFILES_DIR"

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidName reports whether s can name a profile: lower-case letters, digits, dot, dash, underscore.
func ValidName(s string) bool { return nameRE.MatchString(s) }

// Field is one value a tool needs from a profile.
type Field struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// Hidden marks a confidential value: the form masks it.
	Hidden bool `json:"hidden"`
	// Optional fields do not make a profile "not configured" when absent.
	Optional bool `json:"optional,omitempty"`
	// Env names the environment variable that overrides this field, if any.
	Env string `json:"-"`
}

// Dir is DATUM_PROFILES_DIR, else ~/.local/share/datum/profiles (next to the issued secrets).
func Dir() (string, error) {
	if d := os.Getenv(EnvDir); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "datum", "profiles"), nil
}

// Path is the file of a profile.
func Path(name string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("%q is not a valid profile name (lower-case letters, digits, dot, dash, underscore)", name)
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

// Load reads a profile. A profile that does not exist is empty, not an error: the caller decides
// whether that is a problem. A file that anyone but its owner can read is refused.
func Load(name string) (map[string]string, error) {
	path, err := Path(name)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("profile %q (%s) can be read by other users: chmod 600 it", name, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("profile %q (%s) is not a JSON object of strings: %w", name, path, err)
	}
	return values, nil
}

// Save merges values into the profile and writes it atomically with mode 0600 (directory 0700).
// A key with an empty value is removed.
func Save(name string, values map[string]string) error {
	path, err := Path(name)
	if err != nil {
		return err
	}
	current, err := Load(name)
	if err != nil {
		return err
	}
	for k, v := range values {
		if v == "" {
			delete(current, k)
		} else {
			current[k] = v
		}
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating profiles directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, name+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// NotConfiguredError says which fields of a profile have no value. Its text is written for a model:
// it names the call that asks the user, with the fields to pass. It never contains a value.
type NotConfiguredError struct {
	Profile string
	Missing []Field
	// Either is true when any one of the fields would do (a choice such as a DSN or a file).
	Either bool
}

func (e *NotConfiguredError) Error() string {
	fields, _ := json.Marshal(e.Missing)
	var names []string
	for _, f := range e.Missing {
		names = append(names, f.Key)
	}
	which := "lacks " + strings.Join(names, ", ")
	if e.Either {
		which = "needs one of " + strings.Join(names, ", ")
	}
	env := ""
	for _, f := range e.Missing {
		if f.Env != "" {
			env += " " + f.Env
		}
	}
	if env != "" {
		env = " (or set in the server's environment:" + env + ")"
	}
	return fmt.Sprintf("not configured: profile %q %s%s. Ask the user to supply it without it entering the chat: call configure_tool with profile %q and fields %s",
		e.Profile, which, env, e.Profile, fields)
}

// Resolve returns a value for each field: the environment variable if set and non-empty, else the
// profile's. The sources map says where each came from ("environment" or "profile"). Required
// fields that have no value are reported together in a NotConfiguredError. Optional fields that are
// absent are left out of the result.
func Resolve(name string, fields []Field) (values map[string]string, sources map[string]string, err error) {
	profile, err := Load(name)
	if err != nil {
		return nil, nil, err
	}
	values, sources = map[string]string{}, map[string]string{}
	var missing []Field
	for _, f := range fields {
		if f.Env != "" {
			if v := strings.TrimSpace(os.Getenv(f.Env)); v != "" {
				values[f.Key], sources[f.Key] = v, "environment"
				continue
			}
		}
		if v := profile[f.Key]; v != "" {
			values[f.Key], sources[f.Key] = v, "profile"
			continue
		}
		if !f.Optional {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		sort.SliceStable(missing, func(i, j int) bool { return missing[i].Key < missing[j].Key })
		return values, sources, &NotConfiguredError{Profile: name, Missing: missing}
	}
	return values, sources, nil
}

// EnvSecretsDir is the admin tools' directory for issued secrets (mcp-admin writes it; the name is shared here only so the
// file tools can refuse it).
const EnvSecretsDir = "DATUM_ADMIN_SECRETS_DIR"

// ProtectedDirs are the folders that hold credentials the user, not a model, supplies or that a tool issued: the profiles
// directory and the issued-secrets directory.
func ProtectedDirs() []string {
	var dirs []string
	if d, err := Dir(); err == nil {
		dirs = append(dirs, d)
	}
	if d := os.Getenv(EnvSecretsDir); d != "" {
		dirs = append(dirs, d)
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "datum", "secrets"))
	}
	return dirs
}

// canonical resolves symlinks as far as the path exists, so a link into a protected folder is seen for what it is, and a path
// that does not exist yet (a write target) is judged by where it would be.
func canonical(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	rest := ""
	cur := abs
	for {
		if real, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(real, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// Protected reports whether p is, or lies inside, a protected folder. File tools call it on every path they are given and
// refuse the path, whatever workspace roots are open: the values in those files are for the program that reads them, and a
// file read would carry them into a chat.
func Protected(p string) bool {
	c := canonical(p)
	for _, d := range ProtectedDirs() {
		pd := canonical(d)
		if c == pd || strings.HasPrefix(c, pd+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
