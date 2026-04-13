###############################################################################
# Go development tool management
#
# Installs pinned Go tools to ci/tools/bin/ via GOBIN.
# Versions are declared in .config.mk.
#
# Targets:
#   tools-install  — install/update all tools (skips up-to-date ones)
#   tools-tidy     — sync ci/tools/go/go.mod versions from .config.mk
#   tools-clean    — remove ci/tools/bin/
###############################################################################

TOOLS_BIN := $(_REPO_ROOT)/ci/tools/bin

# Prepend tools bin to PATH so go:generate directives and scripts find them.
export PATH := $(TOOLS_BIN):$(PATH)

# Go tool package paths
_PKG_CONTROLLER_GEN := sigs.k8s.io/controller-tools/cmd/controller-gen
_PKG_MOCKGEN        := go.uber.org/mock/mockgen
_PKG_GOLANGCI_LINT  := github.com/golangci/golangci-lint/v2/cmd/golangci-lint
_PKG_GOFUMPT        := mvdan.cc/gofumpt
_PKG_GOVULNCHECK    := golang.org/x/vuln/cmd/govulncheck

_TOOL_ENSURE := GOBIN=$(TOOLS_BIN) $(_REPO_ROOT)/ci/scripts/tool-ensure-go.sh

.PHONY: tools-install
tools-install: | $(TOOLS_BIN)
	@echo "==> Installing Go tools to $(TOOLS_BIN)"
	@$(_TOOL_ENSURE) controller-gen $(_PKG_CONTROLLER_GEN) $(CONTROLLER_GEN_VERSION)
	@$(_TOOL_ENSURE) mockgen        $(_PKG_MOCKGEN)        $(MOCKGEN_VERSION)
	@$(_TOOL_ENSURE) golangci-lint  $(_PKG_GOLANGCI_LINT)  $(GOLANGCI_LINT_VERSION)
	@$(_TOOL_ENSURE) gofumpt        $(_PKG_GOFUMPT)        $(GOFUMPT_VERSION)
	@$(_TOOL_ENSURE) govulncheck    $(_PKG_GOVULNCHECK)    $(GOVULNCHECK_VERSION)
	@echo "==> Done"

.PHONY: tools-tidy
tools-tidy:
	@echo "==> Syncing ci/tools/go/go.mod with .config.mk versions"
	cd $(_REPO_ROOT)/ci/tools/go && go get \
	  $(_PKG_CONTROLLER_GEN)@$(CONTROLLER_GEN_VERSION) \
	  $(_PKG_MOCKGEN)@$(MOCKGEN_VERSION) \
	  $(_PKG_GOLANGCI_LINT)@$(GOLANGCI_LINT_VERSION) \
	  $(_PKG_GOFUMPT)@$(GOFUMPT_VERSION) \
	  $(_PKG_GOVULNCHECK)@$(GOVULNCHECK_VERSION)
	cd $(_REPO_ROOT)/ci/tools/go && go mod tidy
	@echo "==> Done"

.PHONY: tools-clean
tools-clean:
	rm -rf $(TOOLS_BIN)
	@echo "Removed $(TOOLS_BIN)"

$(TOOLS_BIN):
	mkdir -p $@
