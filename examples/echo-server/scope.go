package main

import "github.com/digital-michael/mcp-common/registry"

// BuildScope and AllowedTiers are what bootstrap.Run needs from every server. A real consumer declares them in a
// build-tagged file (mcp-local's cmd/server/scope_local.go is the model); this example only ever serves the local tier.
const BuildScope = "local"

var AllowedTiers = []registry.Tier{registry.TierLocal}
