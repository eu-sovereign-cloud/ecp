include .config.mk
include .common.mk

# CMD is optional: if set, run that command instead of opening an interactive shell.
#   make sh                    # interactive bash
#   make sh CMD="go work sync && git diff go.work"
#   make sh-ctzd CMD="go work sync && git diff go.work"
CMD ?=
.PHONY: sh
sh:
	@if [ -n "$(CMD)" ]; then bash -c '$(CMD)'; else /bin/bash; fi

# Generic variable printer — used by ci/scripts/devcontainer-init.sh to
# resolve computed image names without re-parsing .config.mk in shell.
# Usage: make -s print-TOOLS_IMAGE
.PHONY: print-%
print-%:
	@echo $($*)

###############################################################################
# Git submodules
###############################################################################

.PHONY: submodules
submodules:
	@git submodule sync
	@git submodule update --init --recursive

###############################################################################
# Git hooks
###############################################################################

.PHONY: hooks-install
hooks-install:
	@git -C $(_REPO_ROOT) config core.hooksPath .githooks
	@echo "==> Git hooks installed (core.hooksPath = .githooks)"

.PHONY: hooks-skip-pre-commit
hooks-skip-pre-commit:
	@git -C $(_REPO_ROOT) config hooks.skipPreCommit true
	@echo "==> pre-commit hook disabled (hooks.skipPreCommit = true)"

.PHONY: hooks-unskip-pre-commit
hooks-unskip-pre-commit:
	@git -C $(_REPO_ROOT) config --unset hooks.skipPreCommit || true
	@echo "==> pre-commit hook enabled"

.PHONY: hooks-skip-pre-merge
hooks-skip-pre-merge:
	@git -C $(_REPO_ROOT) config hooks.skipPreMerge true
	@echo "==> pre-push hook disabled (hooks.skipPreMerge = true)"

.PHONY: hooks-unskip-pre-merge
hooks-unskip-pre-merge:
	@git -C $(_REPO_ROOT) config --unset hooks.skipPreMerge || true
	@echo "==> pre-push hook enabled"

###############################################################################
# go-sdk: atomic bump and version-sync verification
#
# The go-sdk source is consumed two ways:
#   1. As a git submodule at modules/go-sdk, used by CRD generation.
#   2. As a Go module dependency declared in gateway/go.mod.
# Both must point at the same upstream tag — otherwise the generated CRDs and
# the compiled types drift apart, and code reviewers can no longer trust that
# the schemas in the repo match what the binary actually understands.
#
# Usage:
#   make go-sdk-update VERSION=v0.5.0    # update both submodule and every go.mod
#   make go-sdk-verify                 # CI gate: fail if they disagree
#   make go-sdk-update-ctzd VERSION=...  # via tools container
#
# go-sdk-update is intentionally an all-or-nothing operation: it checks out the
# submodule at VERSION, then updates each matching go.mod with `go mod edit
# -require=...@VERSION` and runs `go mod download` for the bumped module. If
# either step fails the working tree is left mid-bump for the developer to
# inspect — there is no partial-rollback magic.
#
# go-sdk-verify is wired into pre-commit and pre-merge so a single-place edit
# (bump submodule but forget go.mod, or vice versa) is caught before merge.
###############################################################################

VERSION ?=

.PHONY: go-sdk-update
go-sdk-update:
	@[ -n "$(VERSION)" ] || { echo "error: set VERSION=<tag> (e.g. v0.5.0)"; exit 2; }
	@echo "==> bumping go-sdk to $(VERSION)"
	git -C $(_REPO_ROOT)/modules/go-sdk fetch --tags
	git -C $(_REPO_ROOT)/modules/go-sdk checkout $(VERSION)
	@# Walk every tracked go.mod that requires go-sdk and update it. Matches the
	@# git ls-files enumeration in go-sdk-version-check.sh, so both sides stay in
	@# lock-step as modules are added or removed. Using git ls-files means
	@# .gitignore'd paths (e.g. .cache/go/pkg/mod/...) are excluded for free.
	@#
	@# We use `go mod edit` + `go mod download` instead of `go get` because
	@# `go get` builds the full module graph and tries to fetch
	@# framework@v0.0.1 from the proxy — that pseudo-version only
	@# resolves via the workspace-level replace, which `go get` ignores. The
	@# edit+download combo pins the require and populates go.sum for just the
	@# bumped package, which is all we need.
	@for gomod in $$(grep -l "eu-sovereign-cloud/go-sdk" \
	    $$(git -C $(_REPO_ROOT) ls-files -- '**/go.mod' 'go.mod' | sed "s|^|$(_REPO_ROOT)/|")); do \
	  dir=$$(dirname $$gomod); \
	  echo "==> $$dir"; \
	  (cd $$dir && \
	    go mod edit -require=github.com/eu-sovereign-cloud/go-sdk@$(VERSION) && \
	    go mod download github.com/eu-sovereign-cloud/go-sdk) || exit 1; \
	done
	@echo "==> go work vendor"
	cd $(_REPO_ROOT) && go work vendor
	@echo ""
	@echo "go-sdk bumped to $(VERSION) — review with 'git status' and commit"

.PHONY: go-sdk-verify
go-sdk-verify:
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh go-sdk-verify "go-sdk submodule and go.mod in sync" -- \
	  $(_REPO_ROOT)/ci/scripts/go-sdk-version-check.sh $(_REPO_ROOT)


###############################################################################
# Per-module vulnerability check (govulncheck)
#
# Usage:
#   make framework-vuln          # single module
#   make vuln                                 # all GO_MODULES (parallelisable: -jN)
#   make framework-vuln-ctzd     # via tools container
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
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-vuln" "Vulnerability scan" -- \
	  sh -c "cd $(_REPO_ROOT)/$* && govulncheck ./..."

.PHONY: vuln
vuln: $(addsuffix -vuln,$(GO_MODULES))

###############################################################################
# Per-module tests
#
# Usage:
#   make framework-test                     # all tests, one module
#   make test                                            # all modules
#   make test RUN=TestCreateFoo                          # filter by name
#   make test RUN='TestFoo|TestBar'                      # regex (quote to protect from shell)
#   make framework-test-ctzd RUN=TestFoo    # via tools container
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
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-test" "Unit tests" -- \
	  sh -c "cd $(_REPO_ROOT)/$* && go test -race -v $(if $(RUN),-run '$(RUN)') ./..."

.PHONY: test
test: $(addsuffix -test,$(GO_MODULES))

# test-envtest: run integration tests that require a real kube-apiserver (envtest).
# Set KUBEBUILDER_ASSETS to a directory containing kube-apiserver + etcd binaries,
# or leave it unset to let envtest download them automatically.
.PHONY: test-envtest
test-envtest:
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "resource-envtest" "Envtest integration tests" -- \
	  sh -c "cd $(_REPO_ROOT)/resource && go test -race -tags envtest -v $(if $(RUN),-run '$(RUN)') ./..."

###############################################################################
# Per-module vet-integration: compile-check //go:build integration test files
#
# Usage:
#   make framework-vet-integration
#   make vet-integration
#   make framework-vet-integration-ctzd   # via tools container
#
# Runs `go vet -tags integration ./...` per module. Unlike the plain `test`
# target, this builds files gated by the `integration` build tag — catching
# stale imports and wrong types without executing TestMain (no KIND needed).
# Wired into pre-commit and pre-merge to prevent this class of breakage from
# reaching CI undetected.
###############################################################################

.PHONY: %-vet-integration
%-vet-integration:
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-vet-integration" "Integration build check" -- \
	  sh -c "cd $(_REPO_ROOT)/$* && go vet -tags integration ./..."

.PHONY: vet-integration
vet-integration: $(addsuffix -vet-integration,$(GO_MODULES))

###############################################################################
# Per-module lint (golangci-lint)
#
# Usage:
#   make framework-lint
#   make lint
#   make framework-lint-ctzd
#
# Uses the pinned golangci-lint from ci/tools/bin/ (via tools-install).
# Workspace mode is kept so cross-module replaces resolve correctly.
###############################################################################

.PHONY: %-lint
%-lint: tools-install
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-lint" "Lint" -- \
	  sh -c "cd $(_REPO_ROOT)/$* && golangci-lint run --timeout 10m0s ./..."

.PHONY: lint
lint: $(addsuffix -lint,$(GO_MODULES))

###############################################################################
# Per-module gofmt (golangci-lint fmt)
#
# Usage:
#   make gateway-gofmt              # auto-fix one module
#   make gofmt                                 # auto-fix all modules
#   make gateway-gofmt-check        # check one module (fails on diff)
#   make gofmt-check                           # check all modules (CI gate)
#   make gateway-gofmt-ctzd         # via tools container
#   make gateway-gofmt-check-ctzd   # via tools container
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
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-gofmt-check" "Format check" -- \
	  $(_REPO_ROOT)/ci/scripts/gofmt-check.sh $(_REPO_ROOT)/$* $*

.PHONY: gofmt-check
gofmt-check: $(addsuffix -gofmt-check,$(GO_MODULES))

###############################################################################
# Per-module gosec
#
# Usage:
#   make framework-gosec
#   make gosec
#   make framework-gosec-ctzd
#
# Runs with the Go workspace active (go.work) so that cross-module imports
# resolve correctly. The pinned gosec binary (GOSEC_VERSION in .config.mk)
# lives in ci/tools/bin/, installed by the tools-install prerequisite.
###############################################################################

.PHONY: %-gosec
%-gosec: tools-install
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh "$*-gosec" "Security scan" -- \
	  sh -c "cd $(_REPO_ROOT)/$* && gosec ./..."

.PHONY: gosec
gosec: $(addsuffix -gosec,$(GO_MODULES))

###############################################################################
# Generate API artifacts (CRDs + typed models)
#
# Usage:
#   make generate-api          # run directly on host
#   make generate-api-ctzd     # run inside the tools container
#
# Delegates to framework/backend/kubernetes/Makefile. Kept as a top-level alias so
# CI and developers share one entry point — and so the %-ctzd wrapper composes
# for free.
#
###############################################################################

.PHONY: generate-api
generate-api:
	$(MAKE) -C $(_REPO_ROOT)/framework/backend/kubernetes generate-all

# generate-api-verify — same as generate-api but fails if the tree is dirty
# afterwards. This is what CI runs; developers use `generate-api` directly.
# Mirrors the workspace-verify pattern so both targets stay consistent.
.PHONY: generate-api-verify
generate-api-verify: generate-api
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh generate-api-verify "Generated API artifacts are in sync" -- \
	  $(_REPO_ROOT)/ci/scripts/git-tree-clean-verify.sh --against-index $(_REPO_ROOT) generate-api "make generate-api" framework/backend/kubernetes/ resource/ chart/

###############################################################################
# Per-module: go mod tidy
#
# Usage:
#   make gateway-tidy          # single module
#   make tidy                             # all GO_MODULES
#   make gateway-tidy-ctzd     # inside tools container
#
# CAVEAT: `go mod tidy` intentionally ignores go.work, so it runs the module
# in single-module mode. If a module imports packages from another workspace
# member and relies on go.work's `replace (...)` block to resolve them (as
# csp/aruba does against gateway and framework),
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
#   make gateway-go-get PKG=github.com/foo/bar@v1.2.3
#   make gateway-go-get-ctzd PKG=github.com/foo/bar@v1.2.3
#
# The umbrella `go-get` target runs the same PKG in every GO_MODULE — useful for
# bumping a shared dependency across the workspace in one shot:
#   make go-get PKG=k8s.io/apimachinery@v0.35.0
#
# NOTE: We deliberately use `go mod edit -require=…@VERSION` + `go mod download`
# instead of `go get` + `go mod tidy`. Using `go get` or `go mod tidy` rebuilds
# the full module graph and tries to fetch sibling workspace modules (e.g.
# framework@v0.0.1) from the module proxy — pseudo-versions that
# only resolve via the workspace `replace` directives, which the proxy never
# has. The edit+download approach updates go.mod and go.sum for the target
# package without touching the rest of the module graph.
# The umbrella `go-get` target finishes with `go work sync` to reconcile go.work.sum.
###############################################################################

PKG ?=

.PHONY: %-go-get
%-go-get:
	@[ -n "$(PKG)" ] || { echo "error: set PKG=<module[@version]>"; exit 2; }
	@echo "==> go get $(PKG): $*"
	cd $(_REPO_ROOT)/$* && go mod edit -require=$(PKG) && go mod download $(PKG)

.PHONY: go-get
go-get: $(addsuffix -go-get,$(GO_MODULES))
	@echo "==> go work sync"
	cd $(_REPO_ROOT) && go work sync

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
	cd $(_REPO_ROOT) && go work sync && go list -m all

.PHONY: workspace-verify
workspace-verify: workspace-sync
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh workspace-verify "Go workspace is in sync" -- \
	  $(_REPO_ROOT)/ci/scripts/git-tree-clean-verify.sh --against-head $(_REPO_ROOT) workspace-sync "make workspace-sync" go.work go.work.sum

###############################################################################
# GitHub CLI token provisioning
#
#   make gh-token-ensure                # validate or re-authenticate
#   make gh-token-ensure-ctzd           # inside tools container (has gh)
#
# Stores the token in .cache/gh-token so it is available to both the host
# and tools container (the .cache/ directory is volume-mounted by -ctzd).
# GH_TOKEN is auto-loaded from this file when not set in the environment.
#
# The recipe validates the cached token with `gh api /user`. If the token is
# missing or expired it triggers `gh auth login` (interactive) then persists
# the fresh token.
###############################################################################

_GH_TOKEN_FILE := $(_REPO_ROOT)/.cache/gh-token

# Auto-load GH_TOKEN from cached file when not already set in the environment.
GH_TOKEN ?= $(shell cat $(_GH_TOKEN_FILE) 2>/dev/null)
export GH_TOKEN

.PHONY: gh-token-ensure
gh-token-ensure:
	@$(_REPO_ROOT)/ci/scripts/gh-token-ensure.sh $(_GH_TOKEN_FILE) > /dev/null

###############################################################################
# Verify the current branch is rebased onto its PR target
#
#   make branch-rebase-verify                 # discovers base via `gh pr view`
#   BASE_REF=main make branch-rebase-verify   # explicit (what CI uses)
#   make branch-rebase-verify-ctzd            # via tools container (has gh)
#
# Fails if origin/<base> has commits not in HEAD — the branch must be rebased.
# CI sets BASE_REF from the pull_request event so no gh auth is needed there.
# Locally, falls back to `gh pr view` to discover the base branch for HEAD.
#
# Intentionally standalone: dev loops (lint, test) should not require network
# access. The CI workflow wires this in as an explicit Stage 1 gate.
###############################################################################

.PHONY: branch-rebase-verify
branch-rebase-verify:
	@$(_REPO_ROOT)/ci/scripts/verify-run.sh branch-rebase-verify "Branch is rebased onto target" -- \
	  $(_REPO_ROOT)/ci/scripts/branch-rebase-verify.sh $(_REPO_ROOT)

###############################################################################
# Pre-merge: run the full CI check suite locally
#
#   make pre-merge                # on host
#   make pre-merge-ctzd           # inside tools container
#
# Mirrors the CI pipeline stages but runs all modules (no change filtering).
# Prerequisites are ordered to match CI: rebase gate → all verification jobs.
# If any stage fails make stops — same fail-fast behaviour as CI.
###############################################################################

###############################################################################
# Pre-commit: quick local check before committing
#
#   make pre-commit                # on host
#   make pre-commit-ctzd           # inside tools container
#
# Same as pre-merge but skips the branch-rebase and workspace-sync checks,
# which are only meaningful right before pushing / merging.
###############################################################################

.PHONY: pre-commit
pre-commit: go-sdk-verify generate-api-verify test vet-integration lint gofmt-check vuln gosec

.PHONY: pre-merge
pre-merge: gh-token-ensure branch-rebase-verify workspace-verify go-sdk-verify generate-api-verify test vet-integration lint gofmt-check vuln gosec

###############################################################################
# Workspace membership: add / remove a module from go.work
#
# RELPATH is the path relative to the repo root.
#   make workspace-use-add  RELPATH=csp/newthing
#   make workspace-use-drop RELPATH=csp/oldthing
#
# These targets only manipulate the `use (...)` block. The `replace (...)` block
# in go.work pins workspace modules to their local paths at v0.0.1, which is
# load-bearing for %-vuln and %-gosec (both use GOWORK=off). After adding
# a new workspace module, also run:
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
	  -e GOPATH=$(CONTAINER_WORKSPACE)/.cache/go \
	  -e GOCACHE=$(CONTAINER_WORKSPACE)/.cache/go-build \
	  $(if $(GH_TOKEN),-e GH_TOKEN=$(GH_TOKEN)) \
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
