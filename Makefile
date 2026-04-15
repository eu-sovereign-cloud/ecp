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
# Per-module vulnerability check
#
# Usage:
#   make foundation/persistence-govulncheck          # single module
#   make govulncheck                                 # all GO_MODULES (parallelisable: -jN)
#   make foundation/persistence-govulncheck-ctzd     # via tools container
#
# GOWORK=off forces single-module mode so the scan stays scoped to the
# module's own go.mod. Without it, Go walks up to the repo-root go.work and
# enters workspace mode, potentially scanning unrelated packages.
#
# ci/tools/bin/govulncheck is pre-installed (pinned to GOVULNCHECK_VERSION) in
# both the builder and tools images. On a developer machine running targets
# directly, the tools-install prerequisite ensures the binary is present.
###############################################################################

.PHONY: %-govulncheck
%-govulncheck: tools-install
	@echo "==> govulncheck: $*"
	cd $(_REPO_ROOT)/$* && GOWORK=off govulncheck ./...

.PHONY: govulncheck
govulncheck: $(addsuffix -govulncheck,$(GO_MODULES))

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
# Per-module gosec
#
# Usage:
#   make foundation/persistence-gosec
#   make gosec
#   make foundation/persistence-gosec-ctzd
#
# GOWORK=off mirrors %-govulncheck: keeps the scan scoped to the module's own
# go.mod so findings are attributable to that module. The pinned gosec binary
# (GOSEC_VERSION in .config.mk) lives in ci/tools/bin/, installed by the
# tools-install prerequisite.
###############################################################################

.PHONY: %-gosec
%-gosec: tools-install
	@echo "==> gosec: $*"
	cd $(_REPO_ROOT)/$* && GOWORK=off gosec ./...

.PHONY: gosec
gosec: $(addsuffix -gosec,$(GO_MODULES))

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
