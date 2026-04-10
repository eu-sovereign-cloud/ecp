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
RUNNER_BASE_IMAGE?=${BUILDER_BASE_REGISTRY}/${BUILDER_BASE_REGISTRY_PATH}${BUILDER_BASE_FLAVOR}

###############################################################################
# Local container registry
###############################################################################

LOCAL_REGISTRY?=localhost

###############################################################################
# Builder container (codegen, lint, test, build)
# FROM builder base
###############################################################################

BUILDER_REGISTRY?=${LOCAL_REGISTRY}
BUILDER_REGISTRY_PATH?=ecp/builder
BUILDER_FLAVOR?=${BUILDER_BASE_FLAVOR}-go-v${GO_VERSION}
BUILDER_IMAGE?=${BUILDER_REGISTRY}/${BUILDER_REGISTRY_PATH}:${VERSION}${BUILDER_FLAVOR}
BUILDER_DOCKERFILE?=ci/container/builder/Dockerfile

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
