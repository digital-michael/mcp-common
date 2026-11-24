//go:build membership

package main

import "github.com/digital-michael/mcp-common/registry"

// INTENTIONAL-FAILURE: there is no untagged scope file in this package, so a build with no
// scope tag fails to compile (undefined: BuildScope). That is the design (fail closed).
// Do not add a default or fallback file to make it build. Verify with: make test.

const BuildScope = "membership"

// AllowedTiers: Membership is a superset that includes Local access (see
// registry.TierLocal's doc comment).
var AllowedTiers = []registry.Tier{registry.TierLocal, registry.TierMembership}
