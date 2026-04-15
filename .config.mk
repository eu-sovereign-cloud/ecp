###############################################################################
# SECA Project
###############################################################################

# SECA ECP version
VERSION?=v0.1.0-alpha1-preview

###############################################################################
# Go & Tooling
###############################################################################

# Go & toolchain version
GO_VERSION?=1.26.2

# Builder base image
BUILDER_BASE_REGISTRY?=docker.io
BUILDER_BASE_REGISTRY_PATH?=library/golang
BUILDER_BASE_FLAVOR?=-trixie
BUILDER_BASE_IMAGE?=${BUILDER_BASE_REGISTRY}/${BUILDER_BASE_REGISTRY_PATH}:${GO_VERSION}${BUILDER_BASE_FLAVOR}

# Runner base image
RUNNER_BASE_REGISTRY?=gcr.io
RUNNER_BASE_REGISTRY_PATH?=distroless/static
RUNNER_BASE_FLAVOR?=-debian13
RUNNER_BASE_IMAGE?=${RUNNER_BASE_REGISTRY}/${RUNNER_BASE_REGISTRY_PATH}${RUNNER_BASE_FLAVOR}

###############################################################################
# Local container registry
###############################################################################

LOCAL_REGISTRY?=localhost

###############################################################################
# Builder container (codegen, lint, test, build)
# Published to ghcr.io by CI (.github/workflows/builder-publish.yaml).
# Developers and CI pull the image by digest — no local build step needed.
#
# To modify and test the builder itself locally:
#   make builder-build BUILDER_SOURCE=local
#   make tools-build   BUILDER_SOURCE=local
###############################################################################

# Public registry coordinates (CI pushes here; everyone else only pulls).
BUILDER_PUBLIC_REGISTRY ?= ghcr.io
BUILDER_PUBLIC_REPO     ?= eu-sovereign-cloud/ecp-builder

# Source selector: 'remote' (default) pulls the pinned digest from ghcr.io.
# 'local' builds from ci/container/builder/Dockerfile and tags it :local.
BUILDER_SOURCE ?= remote

BUILDER_DOCKERFILE ?= ci/container/builder/Dockerfile

###############################################################################
# Tools container (interactive shell: completion, coloring, vim)
# FROM builder
###############################################################################

TOOLS_REGISTRY?=${LOCAL_REGISTRY}
TOOLS_REGISTRY_PATH?=ecp/tools
TOOLS_FLAVOR?=${BUILDER_BASE_FLAVOR}-go-v${GO_VERSION}
TOOLS_IMAGE?=${TOOLS_REGISTRY}/${TOOLS_REGISTRY_PATH}:${VERSION}${TOOLS_FLAVOR}
TOOLS_DOCKERFILE?=ci/container/tools/Dockerfile

###############################################################################
# Dev container (SSH, editor servers: gopls, neovim)
# FROM tools
###############################################################################

DEV_REGISTRY?=${LOCAL_REGISTRY}
DEV_REGISTRY_PATH?=ecp/dev
DEV_FLAVOR?=${BUILDER_BASE_FLAVOR}-go-v${GO_VERSION}
DEV_IMAGE?=${DEV_REGISTRY}/${DEV_REGISTRY_PATH}:${VERSION}${DEV_FLAVOR}
DEV_DOCKERFILE?=ci/container/dev/Dockerfile
DEV_CONTAINER_NAME?=ecp-dev
DEV_SSH_PORT?=2222

###############################################################################
# Container workspace
###############################################################################

CONTAINER_WORKSPACE?=/workspace

###############################################################################
# Docker CLI (static binary, for docker-in-docker via socket mount)
###############################################################################

DOCKER_CLI_VERSION?=27.5.1

###############################################################################
# KIND and kubectl (for e2e integration tests)
###############################################################################

KIND_VERSION?=v0.29.0
KUBECTL_VERSION?=v1.33.0

###############################################################################
# Go development tools (installed to ci/tools/bin/ via tools-install)
###############################################################################

CONTROLLER_GEN_VERSION ?= v0.20.0
MOCKGEN_VERSION        ?= v0.6.0
GOLANGCI_LINT_VERSION  ?= v2.1.6
GOFUMPT_VERSION        ?= v0.8.0
GOVULNCHECK_VERSION    ?= v1.1.4
