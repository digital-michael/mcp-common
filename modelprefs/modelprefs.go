package modelprefs

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

// EnvFile overrides the settings file's location.
const EnvFile = "DATUM_MODELS_FILE"

// MinContext and MaxContext bound a context size a user may set. Below the minimum a model cannot hold a useful
// prompt; the maximum is far above any model's trained limit and only catches typos.
const (
	MinContext = 512
	MaxContext = 4 * 1024 * 1024
)

// Prefs is the whole file.
type Prefs struct {
	// Aliases maps a label to a model name.
	Aliases map[string]string `json:"aliases,omitempty"`
	// Context maps a model name to the num_ctx the user wants for it.
	Context map[string]int `json:"context,omitempty"`
}

var aliasRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,39}$`)

// Path is the settings file: DATUM_MODELS_FILE, else $XDG_CONFIG_HOME/datum/models.json, else ~/.config/datum/models.json.
func Path() (string, error) {
	if p := os.Getenv(EnvFile); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "datum", "models.json"), nil
}

// Load reads the settings. A missing file is empty settings, not an error.
func Load() (Prefs, error) {
	path, err := Path()
	if err != nil {
		return Prefs{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Prefs{}, nil
	}
	if err != nil {
		return Prefs{}, err
	}
	var p Prefs
	if err := json.Unmarshal(data, &p); err != nil {
		return Prefs{}, fmt.Errorf("%s is not valid model settings: %w", path, err)
	}
	return p, nil
}

// Save writes the settings atomically (mode 0600, directory 0700).
func Save(p Prefs) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if len(p.Aliases) == 0 {
		p.Aliases = nil
	}
	if len(p.Context) == 0 {
		p.Context = nil
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "models.*.tmp")
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

// Resolve returns the model a name stands for: the alias's target when name is an alias, else name itself.
// ok says whether it was an alias. Aliases do not chain.
func (p Prefs) Resolve(name string) (model string, ok bool) {
	if t, found := p.Aliases[name]; found {
		return t, true
	}
	return name, false
}

// ResolveFile is Resolve on the settings file as it is now. A file that cannot be read leaves the name as it was.
func ResolveFile(name string) string {
	p, err := Load()
	if err != nil {
		return name
	}
	m, _ := p.Resolve(name)
	return m
}

// CheckAlias says why label cannot be an alias for target, or nil. isModel reports whether a name is already a real
// model (installed, or routed by the gateway): an alias must not hide one. An alias target may be a real model or another
// name the caller accepts, but never another alias, so a label always resolves in one step.
func (p Prefs) CheckAlias(label, target string, isModel func(string) bool) error {
	switch {
	case !aliasRE.MatchString(label):
		return fmt.Errorf("alias %q must be 1 to 40 lower-case letters, digits, dot, dash or underscore, starting with a letter or digit (no colon or slash: those belong to model names)", label)
	case strings.HasPrefix(label, "datum-"):
		return fmt.Errorf("alias %q starts with datum-, which is reserved for the deployment's own model names", label)
	case strings.TrimSpace(target) == "":
		return fmt.Errorf("alias %q needs a model to stand for", label)
	case label == target:
		return fmt.Errorf("alias %q cannot stand for itself", label)
	case isModel != nil && isModel(label):
		return fmt.Errorf("%q is already a model name; an alias must not hide a real model", label)
	}
	if _, isAlias := p.Aliases[target]; isAlias {
		return fmt.Errorf("%q is itself an alias (for %q); point %q at the model", target, p.Aliases[target], label)
	}
	for l, t := range p.Aliases {
		if t == label && l != label {
			return fmt.Errorf("alias %q is the target of alias %q; aliases do not chain", label, l)
		}
	}
	return nil
}

// SetAlias adds or replaces an alias after checking it.
func (p *Prefs) SetAlias(label, target string, isModel func(string) bool) error {
	if err := p.CheckAlias(label, target, isModel); err != nil {
		return err
	}
	if p.Aliases == nil {
		p.Aliases = map[string]string{}
	}
	p.Aliases[label] = target
	return nil
}

// RemoveAlias deletes an alias and reports whether it existed.
func (p *Prefs) RemoveAlias(label string) bool {
	if _, ok := p.Aliases[label]; !ok {
		return false
	}
	delete(p.Aliases, label)
	return true
}

// AliasesFor lists the labels that stand for a model, sorted.
func (p Prefs) AliasesFor(model string) []string {
	var out []string
	for l, t := range p.Aliases {
		if t == model {
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return out
}

// SetContext records the context size for a model.
func (p *Prefs) SetContext(model string, n int) error {
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("a model name is needed")
	}
	if n < MinContext || n > MaxContext {
		return fmt.Errorf("context %d is outside %d to %d", n, MinContext, MaxContext)
	}
	if p.Context == nil {
		p.Context = map[string]int{}
	}
	p.Context[model] = n
	return nil
}

// ContextFor is the context size the user set for a model (after resolving an alias), and whether one is set.
func (p Prefs) ContextFor(name string) (int, bool) {
	model, _ := p.Resolve(name)
	n, ok := p.Context[model]
	return n, ok
}
