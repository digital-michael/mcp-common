//go:build global

package main

import "github.com/digital-michael/mcp-common/registry"

// INTENTIONAL-FAILURE: there is no untagged scope file in this package, so a build with no
// scope tag fails to compile (undefined: BuildScope). That is the design (fail closed).
// Do not add a default or fallback file to make it build. Verify with: make test.

const BuildScope = "global"

// AllowedTiers: "global" predates the Tier vocabulary's Local/Membership/Admin
// rename and doesn't map onto any one tier -- kept as the demo's third build
// tag for illustration only, treated as the same kitchen-sink set as
// "membership" here (no TierAdmin -- this example has no admin-tier tool).
var AllowedTiers = []registry.Tier{registry.TierLocal, registry.TierMembership}
