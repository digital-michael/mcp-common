//go:build local

package main

// BuildScope is set only by which build tag produced this binary -- see
// scope_federated.go/scope_global.go for the other variants. There is
// deliberately no untagged fallback file: building with no scope tag at all
// is a compile error (undefined: BuildScope), not a silent default.
const BuildScope = "local"
