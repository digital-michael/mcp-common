// Package config loads per-domain enablement from the environment, the same
// MCP_TOOLS_<GROUP> convention mcp-local already uses -- generalized from a
// hardcoded struct field per domain (which needs a code change every time a
// domain is added) into a name-keyed map driven by whatever domain names the
// caller's registry actually has.
package config

import (
	"os"
	"strconv"
	"strings"
)

// LoadEnabled returns an enabled/disabled flag for each of the given domain
// names, read from MCP_TOOLS_<UPPERCASED_DOMAIN>. An unset or malformed
// value defaults to enabled (fail-open) -- matching mcp-local's existing,
// documented behavior for this same env var convention. This is a
// deliberate match to preserve today's operational behavior during
// migration, not an independently chosen default; revisiting fail-open vs.
// fail-closed here is out of scope for this package.
func LoadEnabled(domains []string) map[string]bool {
	enabled := make(map[string]bool, len(domains))
	for _, domain := range domains {
		enabled[domain] = domainEnabled(domain)
	}
	return enabled
}

func domainEnabled(domain string) bool {
	val, ok := os.LookupEnv("MCP_TOOLS_" + strings.ToUpper(domain))
	if !ok {
		return true
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return true // malformed -> fail open, matches mcp-local's existing behavior
	}
	return b
}
