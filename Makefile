# Library packages (registry/config/bootstrap) are tag-agnostic by design --
# examples/echo-server is NOT, it requires exactly one of local/membership/global
# (see examples/echo-server/scope_*.go) and fails to compile with none, on
# purpose. build/test are scoped accordingly rather than running a bare
# `go build ./...`/`go test ./...`, which would break on the example, not
# the library, and be confusing about why.
LIB_PKGS := ./registry/... ./config/... ./bootstrap/... ./buildinfo/...
EXAMPLE  := ./examples/echo-server
BIN      := bin

# VERSION is a plain-text file, bumped by hand. BUILD_ID is a UTC datetime
# stamp computed fresh at every build -- same convention as mcp-local's own
# Makefile. Both are baked into echo-server via -ldflags, targeting
# buildinfo's package path directly (not a per-binary package) -- this is
# the concrete proof that ldflags-across-a-module-boundary actually works,
# before mcp-admin depends on the same mechanism.
BUILDINFO_PKG := github.com/digital-michael/mcp-common/buildinfo
VERSION       := $(shell cat VERSION 2>/dev/null || echo 0.0.0-unversioned)
BUILD_ID      := $(shell date -u +%Y%m%d%H%M%S)
LDFLAGS       := -X $(BUILDINFO_PKG).Version=$(VERSION) -X $(BUILDINFO_PKG).BuildID=$(BUILD_ID)

.DEFAULT_GOAL := build

.PHONY: build
build: ## Build the echo-server example for each scope variant (local/membership/global)
	@mkdir -p $(BIN)
	go build -ldflags "$(LDFLAGS)" -tags local      -o $(BIN)/echo-server-local      $(EXAMPLE)
	go build -ldflags "$(LDFLAGS)" -tags membership -o $(BIN)/echo-server-membership $(EXAMPLE)
	go build -ldflags "$(LDFLAGS)" -tags global     -o $(BIN)/echo-server-global     $(EXAMPLE)
	@echo "Built with version $(VERSION)+$(BUILD_ID)"

.PHONY: test
test: ## go vet + go test -- library packages need no tag, the example is checked under each one
	go vet $(LIB_PKGS)
	go test $(LIB_PKGS)
	go vet -tags local ./...
	go vet -tags membership ./...
	go vet -tags global ./...
	go test -tags local ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN)

.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | sort

# ── Registration (documentation only -- commented out on purpose) ────────────
#
# mcp-common itself has nothing that should be registered: it's a library,
# not a server, and examples/echo-server is a smoke-test demo, not a real
# tool anyone should have permanently in their tool list (see the ledger
# note on task 9c420e6f for the reasoning).
#
# This block exists so a new contributor -- or a real consumer binary built
# on this library -- has a tested, working example of the *next* step,
# instead of having to reverse-engineer it from mcp-local's Makefile. It's
# copied from mcp-local's actual register-claude/install/unregister/status
# targets (same repo family, same conventions) and adapted to register the
# echo-server-local demo binary under an obviously-a-demo name. Uncomment
# the whole block to actually try it against the demo; when you have a real
# binary built on mcp-common, copy this pattern and swap in its own name.
#
# INSTALL_DIR := $(HOME)/.local/bin
# DEMO_NAME   := mcp-common-echo-example
#
# .PHONY: install
# install: build ## Copy the demo binary to $(HOME)/.local/bin so Claude Code can exec it by path
# 	@mkdir -p "$(INSTALL_DIR)"
# 	# Copy to a temp name then rename over the target rather than overwriting
# 	# the destination file's inode in place -- macOS's kernel code-signature
# 	# cache can go stale against an in-place-truncated executable at a path
# 	# that's already been run, killing every future execve() with "embedded
# 	# signature doesn't match attached signature" (SIGKILL, error 2) until
# 	# the inode is replaced. rename() gives a fresh inode and is atomic.
# 	# (Same gotcha, same fix, as mcp-local's own `make install`.)
# 	cp "$(BIN)/echo-server-local" "$(INSTALL_DIR)/$(DEMO_NAME).new"
# 	mv -f "$(INSTALL_DIR)/$(DEMO_NAME).new" "$(INSTALL_DIR)/$(DEMO_NAME)"
# 	@echo "Installed: $(INSTALL_DIR)/$(DEMO_NAME)"
#
# .PHONY: register-claude
# register-claude: install ## Register the demo binary with Claude Code at user scope
# 	@echo "Registering '$(DEMO_NAME)' with Claude Code at user scope..."
# 	@claude mcp remove "$(DEMO_NAME)" --scope user 2>/dev/null || true
# 	claude mcp add "$(DEMO_NAME)" --scope user -- "$(INSTALL_DIR)/$(DEMO_NAME)"
# 	@echo "Done. Verify with: make status"
#
# .PHONY: unregister
# unregister: ## Remove the demo binary from Claude Code
# 	claude mcp remove "$(DEMO_NAME)" --scope user
# 	@echo "Unregistered '$(DEMO_NAME)' from Claude Code."
#
# .PHONY: status
# status: ## Show Claude Code's MCP server list
# 	claude mcp list
#
# A real consumer binary will likely also want LiteLLM registration --
# mcp-local's `register-litellm` target (calls cortex-stack's
# scripts/register-litellm-mcp.py) is the reference for that; not included
# here since it assumes a cortex-stack checkout this library doesn't need.
