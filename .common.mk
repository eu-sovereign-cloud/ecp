###############################################################################
# Repo root resolution
# Works correctly even under `make -C subdir` because Make resolves the
# include path relative to the invoking Makefile's directory.
###############################################################################

_REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

include $(_REPO_ROOT)/ci/tools/tools.mk

###############################################################################
# Builder image resolution
# BUILDER_IMAGE is resolved here (after _REPO_ROOT is known) based on the
# BUILDER_SOURCE selector set in .config.mk.
#
#   remote (default): use the pinned digest from .builder-digest.
#   local:            use a locally-built image tagged :local.
#
# When .builder-digest is absent or empty (pre-first-publish), falls back to
# :local so the repo is immediately usable without a remote pull.
###############################################################################

_BUILDER_PUBLIC_IMAGE := $(BUILDER_PUBLIC_REGISTRY)/$(BUILDER_PUBLIC_REPO)
_BUILDER_DIGEST_FILE  := $(_REPO_ROOT)/.builder-digest
_BUILDER_DIGEST       := $(strip $(shell [ -r $(_BUILDER_DIGEST_FILE) ] && cat $(_BUILDER_DIGEST_FILE)))

ifeq ($(BUILDER_SOURCE),local)
  BUILDER_IMAGE ?= $(_BUILDER_PUBLIC_IMAGE):local
else ifneq ($(_BUILDER_DIGEST),)
  BUILDER_IMAGE ?= $(_BUILDER_PUBLIC_IMAGE)@$(_BUILDER_DIGEST)
else
  # .builder-digest missing or empty — fall back to local until first CI publish.
  BUILDER_IMAGE ?= $(_BUILDER_PUBLIC_IMAGE):local
  _BUILDER_FALLBACK_LOCAL := 1
endif

###############################################################################
# Enable BuildKit when the buildx component is available.
# The legacy builder stats every file during the context walk before applying
# .dockerignore, so it fails on permission-restricted paths (e.g. .ssh).
# BuildKit applies .dockerignore first and is required for # syntax directives.
###############################################################################

ifneq ($(shell docker buildx version 2>/dev/null),)
  export DOCKER_BUILDKIT := 1
endif

###############################################################################
# Container backend detection (delegated to ci/scripts)
# The command is always `docker`, but the backend may be podman (via
# podman-docker). The backend determines the correct flags for user mapping
# and SELinux volume labeling.
###############################################################################

_CTZD_BACKEND := $(shell $(_REPO_ROOT)/ci/scripts/container-runtime-detect.sh)

###############################################################################
# User mapping and SELinux volume flags (delegated to ci/scripts)
###############################################################################

_CTZD_USER_FLAGS   := $(shell $(_REPO_ROOT)/ci/scripts/container-user-flags.sh $(_CTZD_BACKEND))
_CTZD_SELINUX_OPT  := $(shell $(_REPO_ROOT)/ci/scripts/container-volume-opts.sh $(_CTZD_BACKEND))

# Compose volume option suffixes: ":Z" or "" for plain mounts,
# ":ro,Z" or ":ro" for read-only mounts.
ifneq ($(_CTZD_SELINUX_OPT),)
  _CTZD_VOLUME_OPTS    := :$(_CTZD_SELINUX_OPT)
  _CTZD_VOLUME_OPTS_RO := :ro,$(_CTZD_SELINUX_OPT)
else
  _CTZD_VOLUME_OPTS    :=
  _CTZD_VOLUME_OPTS_RO := :ro
endif

###############################################################################
# Host container socket (for docker-in-docker via socket mount)
###############################################################################

_CTZD_SOCKET        := $(if $(HOST_SOCKET),$(HOST_SOCKET),$(shell $(_REPO_ROOT)/ci/scripts/container-socket-path.sh $(_CTZD_BACKEND)))
_CTZD_SECURITY_OPTS := $(shell $(_REPO_ROOT)/ci/scripts/container-security-opts.sh $(_CTZD_BACKEND))

###############################################################################
# Container run flags (for ephemeral %-ctzd targets)
###############################################################################

# When running nested docker commands from inside a container (DinD via socket
# mount), the volume source path must be the HOST path, not the container path.
# HOST_WORKSPACE is set by ctzdev-start so nested make calls use the real path.
_CTZD_HOST_ROOT := $(if $(HOST_WORKSPACE),$(HOST_WORKSPACE),$(_REPO_ROOT))

# Podman rootless mounts a restricted cgroupfs view inside the container even
# with --cgroupns=host. KIND checks the host cgroup delegation by reading
# /sys/fs/cgroup/user.slice/user-UID.slice/user@UID.service/cgroup.controllers.
# Bind-mounting the user.slice directory makes that path readable inside the
# container so KIND's rootless preflight check passes.
# Prerequisite: host must have systemd cgroup delegation enabled (default on
# Fedora 33+). If the check still fails, run on the host:
#   sudo mkdir -p /etc/systemd/system/user@.service.d/
#   printf '[Service]\nDelegate=yes\n' | sudo tee /etc/systemd/system/user@.service.d/delegate.conf
#   sudo systemctl daemon-reload && loginctl enable-linger $(whoami)
ifeq ($(_CTZD_BACKEND),podman)
  _CTZD_CGROUP_FLAGS := --cgroupns=host -v /sys/fs/cgroup/user.slice:/sys/fs/cgroup/user.slice:ro
else
  _CTZD_CGROUP_FLAGS :=
endif

# On Docker the socket is root:docker (0660); the container user has no docker
# group membership. Pass --group-add with the socket's runtime GID so nested
# docker/kind commands work. On Podman the socket is user-owned via userns, so
# no extra group is needed.
ifeq ($(_CTZD_BACKEND),docker)
  _CTZD_DOCKER_GROUP := --group-add=$(shell stat -c '%g' $(_CTZD_SOCKET) 2>/dev/null)
else
  _CTZD_DOCKER_GROUP :=
endif

_CTZD_RUN_FLAGS := \
  --rm \
  -it \
  --net=host \
  $(_CTZD_USER_FLAGS) \
  $(_CTZD_DOCKER_GROUP) \
  $(_CTZD_SECURITY_OPTS) \
  $(_CTZD_CGROUP_FLAGS) \
  -v $(_CTZD_HOST_ROOT):$(CONTAINER_WORKSPACE)$(_CTZD_VOLUME_OPTS) \
  -v $(_CTZD_SOCKET):/var/run/docker.sock \
  -w $(CONTAINER_WORKSPACE) \
  -e HOME=$(CONTAINER_WORKSPACE)/.cache/container-home \
  -e HOST_WORKSPACE=$(_CTZD_HOST_ROOT) \
  -e HOST_SOCKET=$(_CTZD_SOCKET) \
  -e GOPATH=$(CONTAINER_WORKSPACE)/.cache/go \
  -e GOCACHE=$(CONTAINER_WORKSPACE)/.cache/go-build \
  -e KIND_EXPERIMENTAL_PROVIDER=docker

###############################################################################
# Image build targets (3-layer chain: builder -> tools -> dev)
# _DOCKER_BUILD_FLAGS is injected by *-rebuild targets to pass --no-cache
###############################################################################

_DOCKER_BUILD_FLAGS ?=

.PHONY: builder-build
builder-build:
ifneq ($(BUILDER_SOURCE),local)
	$(error builder-build requires BUILDER_SOURCE=local. The builder image is normally \
	published by CI and pulled automatically. Only rebuild it when modifying \
	ci/container/builder/. Run: make builder-build BUILDER_SOURCE=local)
endif
	docker build $(_DOCKER_BUILD_FLAGS) \
	  --build-arg BUILDER_BASE_IMAGE=$(BUILDER_BASE_IMAGE) \
	  -t $(BUILDER_IMAGE) \
	  -f $(_REPO_ROOT)/$(BUILDER_DOCKERFILE) \
	  $(_REPO_ROOT)

.PHONY: tools-build
tools-build: _builder-ensure-image
	docker build $(_DOCKER_BUILD_FLAGS) \
	  --build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
	  --build-arg DOCKER_CLI_VERSION=$(DOCKER_CLI_VERSION) \
	  --build-arg KIND_VERSION=$(KIND_VERSION) \
	  --build-arg KUBECTL_VERSION=$(KUBECTL_VERSION) \
	  -t $(TOOLS_IMAGE) \
	  -f $(_REPO_ROOT)/$(TOOLS_DOCKERFILE) \
	  $(_REPO_ROOT)

.PHONY: dev-build
dev-build: _tools-ensure-image
	docker build $(_DOCKER_BUILD_FLAGS) \
	  --build-arg TOOLS_IMAGE=$(TOOLS_IMAGE) \
	  -t $(DEV_IMAGE) \
	  -f $(_REPO_ROOT)/$(DEV_DOCKERFILE) \
	  $(_REPO_ROOT)

# images-build always implies BUILDER_SOURCE=local: it is explicitly a "build
# the entire stack from local sources" target.
.PHONY: images-build
images-build:
	$(MAKE) builder-build tools-build dev-build BUILDER_SOURCE=local

###############################################################################
# Force-rebuild targets (bypass Docker layer cache)
###############################################################################

.PHONY: builder-rebuild
builder-rebuild:
	$(MAKE) builder-build BUILDER_SOURCE=local _DOCKER_BUILD_FLAGS=--no-cache

.PHONY: tools-rebuild
tools-rebuild:
	$(MAKE) tools-build _DOCKER_BUILD_FLAGS=--no-cache

.PHONY: dev-rebuild
dev-rebuild:
	$(MAKE) dev-build _DOCKER_BUILD_FLAGS=--no-cache

.PHONY: images-rebuild
images-rebuild:
	$(MAKE) builder-build tools-build dev-build BUILDER_SOURCE=local _DOCKER_BUILD_FLAGS=--no-cache

###############################################################################
# Ensure images exist; auto-build if missing
###############################################################################

.PHONY: _builder-ensure-image
_builder-ensure-image:
	@if [ -n "$$($(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(BUILDER_IMAGE))" ]; then \
	  exit 0; \
	fi; \
	if [ "$(BUILDER_SOURCE)" = "local" ] || [ -n "$(_BUILDER_FALLBACK_LOCAL)" ]; then \
	  echo "Builder image '$(BUILDER_IMAGE)' not found locally. Building..."; \
	  $(MAKE) builder-build BUILDER_SOURCE=local; \
	else \
	  echo "Builder image '$(BUILDER_IMAGE)' not present locally. Pulling from ghcr.io..."; \
	  docker pull $(BUILDER_IMAGE); \
	fi

.PHONY: _tools-ensure-image
_tools-ensure-image: _builder-ensure-image
	@if [ -z "$$($(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(TOOLS_IMAGE))" ]; then \
	  echo "Tools image '$(TOOLS_IMAGE)' not found. Building..."; \
	  $(MAKE) tools-build; \
	fi

.PHONY: _dev-ensure-image
_dev-ensure-image: _tools-ensure-image
	@if [ -z "$$($(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(DEV_IMAGE))" ]; then \
	  echo "Dev image '$(DEV_IMAGE)' not found. Building..."; \
	  $(MAKE) dev-build; \
	fi

###############################################################################
# Image clean targets: remove individual or all container images.
# Named *-image-clean to avoid clashing with tools-clean in ci/tools/tools.mk
# (which removes the Go dev tools binary directory, a different concern).
# Reverse order (dev -> tools -> builder) avoids dependent-image errors.
###############################################################################

.PHONY: builder-image-clean
builder-image-clean:
	docker rmi $(BUILDER_IMAGE) 2>/dev/null || true

.PHONY: tools-image-clean
tools-image-clean:
	docker rmi $(TOOLS_IMAGE) 2>/dev/null || true

.PHONY: dev-image-clean
dev-image-clean:
	docker rmi $(DEV_IMAGE) 2>/dev/null || true

.PHONY: images-clean
images-clean: dev-image-clean tools-image-clean builder-image-clean

###############################################################################
# %-ctzd: run any make target inside the tools container
# Example: make build-ctzd  ->  docker run ... make build
###############################################################################

%-ctzd: _tools-ensure-image
	docker run \
	  $(_CTZD_RUN_FLAGS) \
	  $(TOOLS_IMAGE) \
	  make $*

###############################################################################
# Persistent dev container helpers
###############################################################################

ifeq ($(_CTZD_BACKEND),podman)
  _CTZD_DEV_USER_FLAGS := --userns=keep-id
  _CTZD_DEV_SSH_USER   := $(shell whoami)
else
  _CTZD_DEV_USER_FLAGS :=
  _CTZD_DEV_SSH_USER   := dev
endif

_CTZD_DEV_RUNNING := $(shell docker ps -q -f name=^$(DEV_CONTAINER_NAME)$$ 2>/dev/null)
