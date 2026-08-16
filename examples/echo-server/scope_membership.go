//go:build membership

package main

import "github.com/digital-michael/mcp-common/registry"

const BuildScope = "membership"

// AllowedTiers: Membership is a superset that includes Local access (see
// registry.TierLocal's doc comment).
var AllowedTiers = []registry.Tier{registry.TierLocal, registry.TierMembership}
