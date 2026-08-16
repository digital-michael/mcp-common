//go:build local

package main

import "github.com/digital-michael/mcp-common/registry"

// BuildScope is set only by which build tag produced this binary -- see
// scope_membership.go/scope_global.go for the other variants. There is
// deliberately no untagged fallback file: building with no scope tag at all
// is a compile error (undefined: BuildScope), not a silent default.
const BuildScope = "local"

// AllowedTiers is this scope's tier allow-list, passed to
// registry.RegisterWithServer via bootstrap.Run -- see that doc comment for
// why each consumer owns this mapping rather than mcp-common inferring it
// from the scope string.
var AllowedTiers = []registry.Tier{registry.TierLocal}
