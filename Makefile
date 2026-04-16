include .config.mk
include .common.mk

.PHONY: sh
sh:
	/bin/bash

# Generic variable printer — used by ci/scripts/devcontainer-init.sh to
# resolve computed image names without re-parsing .config.mk in shell.
# Usage: make -s print-TOOLS_IMAGE
.PHONY: print-%
print-%:
	@echo $($*)

###############################################################################
# Per-module vulnerability check (govulncheck)
#
# Usage:
#   make foundation/persistence-vuln          # single module
#   make vuln                                 # all GO_MODULES (parallelisable: -jN)
#   make foundation/persistence-vuln-ctzd     # via tools container
#
# GOWORK=off forces single-module mode so the scan stays scoped to the
# module's own go.mod. Without it, Go walks up to the repo-root go.work and
# enters workspace mode, potentially scanning unrelated packages.
#
# ci/tools/bin/govulncheck is pre-installed (pinned to GOVULNCHECK_VERSION) in
# both the builder and tools images. On a developer machine running targets
# directly, the tools-install prerequisite ensures the binary is present.
###############################################################################

.PHONY: %-vuln
%-vuln: tools-install
	@echo "==> vuln: $*"
	cd $(_REPO_ROOT)/$* && govulncheck ./...

.PHONY: vuln
vuln: $(addsuffix -vuln,$(GO_MODULES))

###############################################################################
# Per-module tests
#
# Usage:
#   make foundation/persistence-test                     # all tests, one module
#   make test                                            # all modules
#   make test RUN=TestCreateFoo                          # filter by name
#   make test RUN='TestFoo|TestBar'                      # regex (quote to protect from shell)
#   make foundation/persistence-test-ctzd RUN=TestFoo    # via tools container
#
# RUN is optional. When set it is forwarded verbatim to `go test -run <regex>`,
# which matches the test's fully qualified name (Go's own filter semantics —
# see `go help testflag`). The umbrella `test` target propagates RUN to every
# module because all per-module rules run in the same make invocation.
#
# Workspace mode is left enabled (no GOWORK=off): tests depend on the replace
# directives declared in go.work. `cd $(_REPO_ROOT)/$*` scopes the run to the
# module's own package tree.
###############################################################################

RUN ?=

.PHONY: %-test
%-test:
	@echo "==> test: $*$(if $(RUN), (filter: $(RUN)))"
	cd $(_REPO_ROOT)/$* && go test -race -v $(if $(RUN),-run '$(RUN)') ./...

.PHONY: test
test: $(addsuffix -test,$(GO_MODULES))

###############################################################################
# Per-module lint (golangci-lint)
#
# Usage:
#   make foundation/persistence-lint
#   make lint
#   make foundation/persistence-lint-ctzd
#
# Uses the pinned golangci-lint from ci/tools/bin/ (via tools-install).
# Workspace mode is kept so cross-module replaces resolve correctly.
###############################################################################

.PHONY: %-lint
%-lint: tools-install
	@echo "==> lint: $*"
	cd $(_REPO_ROOT)/$* && golangci-lint run --timeout 10m0s ./...

.PHONY: lint
lint: $(addsuffix -lint,$(GO_MODULES))

###############################################################################
# Per-module gofmt (golangci-lint fmt)
#
# Usage:
#   make foundation/gateway-gofmt              # auto-fix one module
#   make gofmt                                 # auto-fix all modules
#   make foundation/gateway-gofmt-check        # check one module (fails on diff)
#   make gofmt-check                           # check all modules (CI gate)
#   make foundation/gateway-gofmt-ctzd         # via tools container
#   make foundation/gateway-gofmt-check-ctzd   # via tools container
#
# Uses `golangci-lint fmt`, which applies the formatters configured in
# .golangci.yml (gofmt simplify + goimports). This keeps `make gofmt` and
# `make lint` in lock-step: whatever lint reports as a formatter violation,
# gofmt fixes — no divergence between the two.
#
# %-gofmt        writes fixes in place (developer convenience).
# %-gofmt-check  runs in --diff mode and exits non-zero if any diff is
#                produced. This is what CI calls so the workspace is never
#                mutated on the runner.
###############################################################################

.PHONY: %-gofmt
%-gofmt: tools-install
	@echo "==> gofmt: $*"
	cd $(_REPO_ROOT)/$* && golangci-lint fmt ./...

.PHONY: gofmt
gofmt: $(addsuffix -gofmt,$(GO_MODULES))

.PHONY: %-gofmt-check
%-gofmt-check: tools-install
	@echo "==> gofmt-check: $*"
	@diff=$$(cd $(_REPO_ROOT)/$* && golangci-lint fmt --diff ./... 2>&1); \
	if [ -n "$$diff" ]; then \
	  printf '%s\n' "$$diff"; \
	  echo "FAIL: $* has unformatted files (run: make $*-gofmt)"; \
	  exit 1; \
	fi

.PHONY: gofmt-check
gofmt-check: $(addsuffix -gofmt-check,$(GO_MODULES))

###############################################################################
# Per-module gosec
#
# Usage:
#   make foundation/persistence-gosec
#   make gosec
#   make foundation/persistence-gosec-ctzd
#
# Runs with the Go workspace active (go.work) so that cross-module imports
# resolve correctly. The pinned gosec binary (GOSEC_VERSION in .config.mk)
# lives in ci/tools/bin/, installed by the tools-install prerequisite.
###############################################################################

.PHONY: %-gosec
%-gosec: tools-install
	@echo "==> gosec: $*"
	cd $(_REPO_ROOT)/$* && gosec ./...

.PHONY: gosec
gosec: $(addsuffix -gosec,$(GO_MODULES))

###############################################################################
# Generate API artifacts (CRDs + typed models)
#
# Usage:
#   make generate-api          # run directly on host
#   make generate-api-ctzd     # run inside the tools container
#
# Delegates to foundation/persistence/Makefile, which is the only module with
# generated artifacts today. Kept as a top-level alias so CI and developers
# share one entry point — and so the %-ctzd wrapper composes for free.
#
# Requires python3 on PATH (used by
# foundation/persistence/scripts/replace-reference-fields.py). The builder
# image installs it; host developers need it in their own PATH.
###############################################################################

.PHONY: generate-api
generate-api:
	$(MAKE) -C $(_REPO_ROOT)/foundation/persistence generate-all

# generate-api-verify — same as generate-api but fails if the tree is dirty
# afterwards. This is what CI runs; developers use `generate-api` directly.
# Mirrors the workspace-verify pattern so both targets stay consistent.
.PHONY: generate-api-verify
generate-api-verify: generate-api
	@if [ -n "$$(cd $(_REPO_ROOT) && git status --porcelain)" ]; then \
	  echo "::error::generate-api produced changes. Please run 'make generate-api' and commit the results."; \
	  cd $(_REPO_ROOT) && git diff; \
	  cd $(_REPO_ROOT) && git status; \
	  exit 1; \
	fi

###############################################################################
# Per-module: go mod tidy
#
# Usage:
#   make foundation/gateway-tidy          # single module
#   make tidy                             # all GO_MODULES
#   make foundation/gateway-tidy-ctzd     # inside tools container
#
# CAVEAT: `go mod tidy` intentionally ignores go.work, so it runs the module
# in single-module mode. If a module imports packages from another workspace
# member and relies on go.work's `replace (...)` block to resolve them (as
# foundation/delegator does against foundation/gateway and foundation/persistence),
# tidy will fail trying to fetch the v0.0.1 pseudo-version from the proxy.
#
# Use this target only on modules whose imports resolve in isolation, or
# after adding the same `replace` line to the module's own go.mod.
#
# For the workspace-level "clean up after dep edits" operation, use
# `make workspace-sync` — it runs `go work sync`, which is the workspace
# equivalent of `go mod tidy`. Vendoring is disabled repo-wide.
###############################################################################

.PHONY: %-tidy
%-tidy:
	@echo "==> mod tidy: $*"
	cd $(_REPO_ROOT)/$* && go mod tidy

.PHONY: tidy
tidy: $(addsuffix -tidy,$(GO_MODULES))

###############################################################################
# Per-module: go get <pkg[@version]>
#
# PKG is required; include @version to pin (or @latest to upgrade).
#   make foundation/gateway-get PKG=github.com/foo/bar@v1.2.3
#   make foundation/gateway-get-ctzd PKG=github.com/foo/bar@v1.2.3
#
# The umbrella `get` target runs the same PKG in every GO_MODULE — useful for
# bumping a shared dependency across the workspace in one shot:
#   make get PKG=k8s.io/apimachinery@v0.35.0
#
# `go mod tidy` is always run after `go get` to keep go.sum consistent.
###############################################################################

PKG ?=

.PHONY: %-get
%-get:
	@[ -n "$(PKG)" ] || { echo "error: set PKG=<module[@version]>"; exit 2; }
	@echo "==> go get $(PKG): $*"
	cd $(_REPO_ROOT)/$* && go get $(PKG) && go mod tidy

.PHONY: get
get: $(addsuffix -get,$(GO_MODULES))

###############################################################################
# Workspace-level: sync and CI verify
#
#   make workspace-sync      # go work sync
#   make workspace-verify    # workspace-sync, then fail if git tree is dirty
#
# Both compose with -ctzd:
#   make workspace-sync-ctzd
#   make workspace-verify-ctzd
#
# `workspace-verify` is what CI runs — the same sync operation plus a
# git-cleanliness gate, so the local fix and the CI check share one code path.
###############################################################################

.PHONY: workspace-sync
workspace-sync:
	@echo "==> go work sync"
	cd $(_REPO_ROOT) && go work sync

.PHONY: workspace-verify
workspace-verify: workspace-sync
	@if [ -n "$$(cd $(_REPO_ROOT) && git status --porcelain)" ]; then \
	  echo "::error::workspace-sync produced changes. Please run 'make workspace-sync' and commit the results."; \
	  cd $(_REPO_ROOT) && git diff; \
	  cd $(_REPO_ROOT) && git status; \
	  exit 1; \
	fi

###############################################################################
# Workspace membership: add / remove a module from go.work
#
# RELPATH is the path relative to the repo root.
#   make workspace-use-add  RELPATH=foundation/plugin/newthing
#   make workspace-use-drop RELPATH=foundation/plugin/oldthing
#
# These targets only manipulate the `use (...)` block. The `replace (...)` block
# in go.work pins foundation modules to their local paths at v0.0.1, which is
# load-bearing for %-vuln and %-gosec (both use GOWORK=off). After adding
# a new foundation module, also run:
#
#   go work edit -replace <modpath>=./$(RELPATH)
#
# and commit the result. This is intentionally left manual — the module path and
# replace convention vary per module. Run `make workspace-sync` when done.
###############################################################################

RELPATH ?=

.PHONY: workspace-use-add
workspace-use-add:
	@[ -n "$(RELPATH)" ] || { echo "error: set RELPATH=<path/to/module>"; exit 2; }
	@[ -f "$(_REPO_ROOT)/$(RELPATH)/go.mod" ] || { echo "error: $(RELPATH)/go.mod not found"; exit 2; }
	@echo "==> go work use ./$(RELPATH)"
	cd $(_REPO_ROOT) && go work use ./$(RELPATH)
	@echo "Reminder: if this module needs a local replace for GOWORK=off scans,"
	@echo "  run: go work edit -replace <modpath>=./$(RELPATH)"
	@echo "Then: make workspace-sync"

.PHONY: workspace-use-drop
workspace-use-drop:
	@[ -n "$(RELPATH)" ] || { echo "error: set RELPATH=<path/to/module>"; exit 2; }
	@echo "==> go work edit -dropuse ./$(RELPATH)"
	cd $(_REPO_ROOT) && go work edit -dropuse ./$(RELPATH)
	@echo "Reminder: also drop the matching replace (if any):"
	@echo "  go work edit -dropreplace <modpath>"
	@echo "Then: make workspace-sync"

###############################################################################
# Persistent dev container
###############################################################################

.PHONY: ctzdev-start
ctzdev-start: _dev-ensure-image
ifneq ($(_CTZD_DEV_RUNNING),)
	@echo "Dev container '$(DEV_CONTAINER_NAME)' is already running."
	@echo "  SSH: ssh -p $(DEV_SSH_PORT) $(_CTZD_DEV_SSH_USER)@localhost"
else
	docker run -d \
	  --name $(DEV_CONTAINER_NAME) \
	  --net=host \
	  $(_CTZD_DEV_USER_FLAGS) \
	  $(_CTZD_SECURITY_OPTS) \
	  $(_CTZD_CGROUP_FLAGS) \
	  -v $(_REPO_ROOT):$(CONTAINER_WORKSPACE)$(_CTZD_VOLUME_OPTS) \
	  -v $(HOME)/.ssh:/tmp/host-ssh$(_CTZD_VOLUME_OPTS_RO) \
	  -v $(_CTZD_SOCKET):/var/run/docker.sock$(_CTZD_VOLUME_OPTS) \
	  -e HOST_UID=$$(id -u) \
	  -e HOST_GID=$$(id -g) \
	  -e HOME=$(CONTAINER_WORKSPACE)/.cache/container-home \
	  -e HOST_WORKSPACE=$(_REPO_ROOT) \
	  -e HOST_SOCKET=$(_CTZD_SOCKET) \
	  -e DEV_SSH_PORT=$(DEV_SSH_PORT) \
	  -e KIND_EXPERIMENTAL_PROVIDER=docker \
	  $(DEV_IMAGE)
	@echo ""
	@echo "Dev container '$(DEV_CONTAINER_NAME)' started."
	@echo "  SSH: ssh -p $(DEV_SSH_PORT) $(_CTZD_DEV_SSH_USER)@localhost"
	@echo ""
endif

.PHONY: ctzdev-stop
ctzdev-stop:
	@docker stop $(DEV_CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(DEV_CONTAINER_NAME) 2>/dev/null || true
	@echo "Dev container '$(DEV_CONTAINER_NAME)' stopped and removed."

.PHONY: ctzdev-status
ctzdev-status:
	@if docker ps -q -f name=^$(DEV_CONTAINER_NAME)$$ 2>/dev/null | grep -q .; then \
	  echo "Dev container '$(DEV_CONTAINER_NAME)' is running."; \
	  echo "  SSH: ssh -p $(DEV_SSH_PORT) $(_CTZD_DEV_SSH_USER)@localhost"; \
	else \
	  echo "Dev container '$(DEV_CONTAINER_NAME)' is not running."; \
	fi
