package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// Class says what a call does. Every tool has exactly one.
type Class string

const (
	// Read changes nothing anywhere.
	Read Class = "read"
	// Write changes this machine's state, or writes files.
	Write Class = "write"
	// Publish sends data off this machine.
	Publish Class = "publish"
)

// Access classifies one tool. It is the single source for the tool's MCP
// annotations and its description prefix, so the two cannot disagree. A tool
// with no annotations at all is presented by MCP clients as destructive and
// open-world, which is wrong for most tools and hides the ones that are.
type Access struct {
	Class Class
	// Destructive: a call can lose data (overwrite, delete, remove access).
	// Meaningless for Read, which changes nothing.
	Destructive bool
	// OpenWorld: the call reaches something that may not be this machine
	// (another host, an external service). Loopback-only calls are not.
	OpenWorld bool
}

// Prefix is the label at the start of the tool's description.
func (a Access) Prefix() string { return "[" + strings.ToUpper(string(a.Class)) + "] " }

// Classify stamps every registered tool's annotations and description prefix
// from table, keyed by tool name. A tool with no entry is an error naming it
// (and every other such tool): classification is fail-closed, so a new tool
// cannot ship unclassified. classifiedElsewhere names tools a consumer already
// classifies itself (for example a toolset with its own table); they are left
// alone. Entries for tools this build does not register are ignored here --
// a build may omit domains -- and are for a test to catch via Unclassified.
func (r *Registry) Classify(table map[string]Access, classifiedElsewhere ...string) error {
	skip := make(map[string]bool, len(classifiedElsewhere))
	for _, n := range classifiedElsewhere {
		skip[n] = true
	}
	var missing []string
	for i := range r.defs {
		name := r.defs[i].Tool.Name
		if skip[name] {
			continue
		}
		a, ok := table[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if a.Class != Read && a.Class != Write && a.Class != Publish {
			return fmt.Errorf("tool %q has unknown access class %q", name, a.Class)
		}
		stamp(&r.defs[i].Tool, a)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("tools not classified in the access table: %s", strings.Join(missing, ", "))
	}
	return nil
}

// Unclassified reports the table entries that match no registered tool (a
// stale or misspelled name) -- for a test over the full tool set.
func (r *Registry) Unclassified(table map[string]Access) []string {
	var out []string
	for name := range table {
		if _, ok := r.byName[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func stamp(t *mcp.Tool, a Access) {
	if !strings.HasPrefix(t.Description, a.Prefix()) {
		t.Description = a.Prefix() + t.Description
	}
	ro := a.Class == Read
	destructive := a.Class != Read && a.Destructive
	open := a.Class == Publish || a.OpenWorld
	t.Annotations.ReadOnlyHint = &ro
	t.Annotations.DestructiveHint = &destructive
	t.Annotations.IdempotentHint = &ro
	t.Annotations.OpenWorldHint = &open
}
