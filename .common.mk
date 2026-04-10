###############################################################################
# Repo root resolution
# Works correctly even under `make -C subdir` because Make resolves the
# include path relative to the invoking Makefile's directory.
###############################################################################

_REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

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

_CTZD_RUN_FLAGS := \
  --rm \
  -it \
  $(_CTZD_USER_FLAGS) \
  $(_CTZD_SECURITY_OPTS) \
  -v $(_CTZD_HOST_ROOT):$(CONTAINER_WORKSPACE)$(_CTZD_VOLUME_OPTS) \
  -v $(_CTZD_SOCKET):/var/run/docker.sock \
  -w $(CONTAINER_WORKSPACE) \
  -e HOME=$(CONTAINER_WORKSPACE)/.cache/container-home \
  -e HOST_WORKSPACE=$(_CTZD_HOST_ROOT) \
  -e HOST_SOCKET=$(_CTZD_SOCKET) \
  -e GOPATH=$(CONTAINER_WORKSPACE)/.cache/go \
  -e GOCACHE=$(CONTAINER_WORKSPACE)/.cache/go-build

###############################################################################
# Image build targets (3-layer chain: builder -> tools -> dev)
###############################################################################

.PHONY: builder-build
builder-build:
	docker build \
	  --build-arg BUILDER_BASE_IMAGE=$(BUILDER_BASE_IMAGE) \
	  -t $(BUILDER_IMAGE) \
	  -f $(_REPO_ROOT)/$(BUILDER_DOCKERFILE) \
	  $(_REPO_ROOT)

.PHONY: tools-build
tools-build: _builder-ensure-image
	docker build \
	  --build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
	  --build-arg DOCKER_CLI_VERSION=$(DOCKER_CLI_VERSION) \
	  -t $(TOOLS_IMAGE) \
	  -f $(_REPO_ROOT)/$(TOOLS_DOCKERFILE) \
	  $(_REPO_ROOT)

.PHONY: dev-build
dev-build: _tools-ensure-image
	docker build \
	  --build-arg TOOLS_IMAGE=$(TOOLS_IMAGE) \
	  -t $(DEV_IMAGE) \
	  -f $(_REPO_ROOT)/$(DEV_DOCKERFILE) \
	  $(_REPO_ROOT)

###############################################################################
# Ensure images exist; auto-build if missing
###############################################################################

_BUILDER_IMAGE_EXISTS := $(shell $(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(BUILDER_IMAGE))
_TOOLS_IMAGE_EXISTS   := $(shell $(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(TOOLS_IMAGE))
_DEV_IMAGE_EXISTS     := $(shell $(_REPO_ROOT)/ci/scripts/container-image-exists.sh $(DEV_IMAGE))

.PHONY: _builder-ensure-image
_builder-ensure-image:
ifeq ($(_BUILDER_IMAGE_EXISTS),)
	@echo "Builder image '$(BUILDER_IMAGE)' not found. Building..."
	@$(MAKE) builder-build
endif

.PHONY: _tools-ensure-image
_tools-ensure-image: _builder-ensure-image
ifeq ($(_TOOLS_IMAGE_EXISTS),)
	@echo "Tools image '$(TOOLS_IMAGE)' not found. Building..."
	@$(MAKE) tools-build
endif

.PHONY: _dev-ensure-image
_dev-ensure-image: _tools-ensure-image
ifeq ($(_DEV_IMAGE_EXISTS),)
	@echo "Dev image '$(DEV_IMAGE)' not found. Building..."
	@$(MAKE) dev-build
endif

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
