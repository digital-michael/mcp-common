// Package bootstrap provides a reusable CLI + MCP-stdio-server entry point
// for a binary built on a registry.Registry, generalizing mcp-local's
// existing main.go harness (-list-tools/-run/-help/default server mode) so
// each new consumer binary doesn't need to reimplement it.
package bootstrap

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/digital-michael/mcp-common/config"
	"github.com/digital-michael/mcp-common/registry"
)

// Run parses CLI flags and either runs the MCP stdio server (default), or
// serves one of the introspection/direct-call modes (-version, -list-tools,
// -help, -run). name/version identify this binary; family/scope are
// stamped into every registered tool's _meta by the registry (see
// registry.Registry.RegisterWithServer); allowedTiers is this binary's own
// tier allow-list for its BuildScope (see registry.RegisterWithServer's doc
// comment -- typically declared alongside BuildScope in the caller's own
// scope_*.go files); domains is the full list of domain names this binary
// knows about, used to load MCP_TOOLS_<GROUP> enablement via
// config.LoadEnabled regardless of which domains actually have any tools
// registered yet.
func Run(name, version, family, scope string, allowedTiers []registry.Tier, reg *registry.Registry, domains []string) {
	showHelp := flag.Bool("help", false, "Show usage, or detailed help for a specific tool (with -help <tool>)")
	showVersion := flag.Bool("version", false, "Print name and version")
	listTools := flag.Bool("list-tools", false, "List every registered tool")
	runTool := flag.String("run", "", "Run a tool directly by name, bypassing the MCP transport")
	flag.Usage = func() { usage(name) }
	flag.Parse()

	positional := flag.Args()

	switch {
	case *showVersion:
		fmt.Printf("%s %s\n", name, version)
	case *listTools:
		printTools(reg)
	case *runTool != "":
		if err := runToolCmd(reg, *runTool, positional); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case *showHelp:
		if len(positional) > 0 {
			printToolHelp(reg, positional[0])
		} else {
			usage(name)
		}
	default:
		runServer(name, version, family, scope, allowedTiers, reg, domains)
	}
}

func runServer(name, version, family, scope string, allowedTiers []registry.Tier, reg *registry.Registry, domains []string) {
	enabled := config.LoadEnabled(domains)
	s := server.NewMCPServer(name, version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	reg.RegisterWithServer(s, family, scope, allowedTiers, enabled)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

func printTools(reg *registry.Registry) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDOMAIN\tTIER\tDESCRIPTION")
	for _, def := range reg.Describe() {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", def.Tool.Name, def.Domain, def.Tier, def.Tool.Description)
	}
	w.Flush()
}

func printToolHelp(reg *registry.Registry, name string) {
	def, ok := reg.FindMeta(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown tool %q — run -list-tools to see available tools\n", name)
		os.Exit(1)
	}
	fmt.Printf("%s — %s\n", def.Tool.Name, def.Tool.Description)
	if def.CLI.Args != "" {
		fmt.Printf("Args: %s\n", def.CLI.Args)
	}
	if len(def.CLI.Examples) > 0 {
		fmt.Println("Examples:")
		for k, v := range def.CLI.Examples {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}
}

// runToolCmd resolves name's ToolDef, maps positional makes CLI arguments
// onto its declared Params (or accepts key=value pairs), enforces
// RequiredN, and invokes it via reg.Run.
func runToolCmd(reg *registry.Registry, name string, args []string) error {
	def, ok := reg.FindMeta(name)
	if !ok {
		return fmt.Errorf("unknown tool %q — run -list-tools to see available tools", name)
	}

	parsed := make(map[string]any)
	if looksLikeKeyValue(args) {
		for _, a := range args {
			k, v, _ := strings.Cut(a, "=")
			parsed[k] = v
		}
	} else {
		for i, v := range args {
			if i >= len(def.CLI.Params) {
				return fmt.Errorf("too many positional args for %q (expects %d)", name, len(def.CLI.Params))
			}
			parsed[def.CLI.Params[i]] = v
		}
	}

	if len(parsed) < def.CLI.RequiredN {
		return fmt.Errorf("%q requires %d argument(s), got %d", name, def.CLI.RequiredN, len(parsed))
	}

	result, err := reg.Run(context.Background(), name, parsed)
	if err != nil {
		return err
	}
	printResult(result)
	return nil
}

func looksLikeKeyValue(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, a := range args {
		if !strings.Contains(a, "=") {
			return false
		}
	}
	return true
}

func printResult(result *mcp.CallToolResult) {
	for _, c := range result.Content {
		if text, ok := mcp.AsTextContent(c); ok {
			fmt.Println(text.Text)
			return
		}
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func usage(name string) {
	fmt.Fprintf(os.Stderr, `%s

Usage:
  %s                    Run as an MCP stdio server
  %s -list-tools         List every registered tool
  %s -run <tool> [args]  Run a tool directly (positional or key=value args)
  %s -help [tool]        Show usage, or detailed help for one tool
  %s -version             Print name and version
`, name, name, name, name, name, name)
}
