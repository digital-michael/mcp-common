//go:build global

package main

import "github.com/digital-michael/mcp-common/registry"

const BuildScope = "global"

// AllowedTiers: "global" predates the Tier vocabulary's Local/Membership/Admin
// rename and doesn't map onto any one tier -- kept as the demo's third build
// tag for illustration only, treated as the same kitchen-sink set as
// "membership" here (no TierAdmin -- this example has no admin-tier tool).
var AllowedTiers = []registry.Tier{registry.TierLocal, registry.TierMembership}
