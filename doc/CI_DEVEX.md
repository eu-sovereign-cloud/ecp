# CI and Developer Experience

This document covers how to set up a development environment, how the Makefile is structured, and how local targets relate to the CI pipeline.

## Overview

ECP uses a **container-first** development model:

- All compilation and tooling run inside container images — **Go is not required on the host**.
- A 3-layer image chain (`builder` → `tools` → `dev`) provides progressively richer environments.
- Every root Makefile target can run either directly on the host or inside the tools container by appending `-ctzd` (e.g., `make test-ctzd`).
- The `pre-commit` and `pre-merge` targets mirror what CI runs, so local failures predict CI failures exactly.

## Prerequisites

| Tool | Purpose | Notes |
|------|---------|-------|
| Docker or Podman | Container runtime | Auto-detected; both are fully supported |
| `kubectl` | Kubernetes CLI | Required for cluster operations |
| KIND | Local Kubernetes clusters | Required for integration tests |
| go 1.26.4+ | Build/test on host | Required only for bare-metal workflow |

> **Podman users:** The Makefile handles SELinux volume labels (`:Z`), cgroupv2 delegation, rootless userns mapping, and KIND preflight automatically. See `.common.mk` for details.

The builder image is published to `ghcr.io/eu-sovereign-cloud/ecp-builder` and pulled automatically on first use. Its pinned digest is stored in `.builder-digest` and updated by an automated CI PR whenever the builder inputs change.

### Registry authentication

The toolchain pulls from `docker.io` (builder base image) and `ghcr.io` (published builder image). Authenticating to both avoids rate-limit failures on first build.

**`docker.io`** — Docker Hub enforces anonymous-pull rate limits. The builder
base image (`golang:1.26.4-trixie`) is fetched from Docker Hub on a local
builder build (`make builder-rebuild`):

```bash
docker login docker.io
```

**`ghcr.io`** — Generate a GitHub **classic** Personal Access Token (PAT) at
*GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)*
with the **`read:packages`** scope (the minimum required to pull container
images; developers only pull — CI pushes). Then log in:

```bash
# Interactive (paste PAT as password)
docker login ghcr.io -u <github-username>

# Non-interactive
echo "<PAT>" | docker login ghcr.io -u <github-username> --password-stdin
```

> **Note:** The `ghcr.io/eu-sovereign-cloud/ecp-builder` image is public, so
> anonymous pulls normally succeed. The PAT login is only needed when anonymous
> pulls fail or hit the GHCR rate limit.

> **Podman users:** substitute `podman login` for `docker login` (same arguments).

## Development Workflows

### Bare-Metal Development

Running directly on the host requires go 1.26.4+ and the dev tools installed locally.

```bash
# Install pinned dev tools to ci/tools/bin/ (golangci-lint, controller-gen, etc.)
make tools-install

# Activate git hooks (pre-commit and pre-push gates)
make hooks-install

# Generate CRD YAML and typed Go models from OpenAPI specs
make generate-api

# Run tests (all modules)
make test

# Run tests for a single module
make foundation/gateway-test

# Filter tests by name
make test RUN=TestCreateWorkspace

# Lint all modules
make lint

# Format all modules in-place
make gofmt

# Vulnerability scan
make vuln

# Security scan
make gosec

# Full local validation gate (mirrors CI, skips branch/workspace checks)
make pre-commit

# Full CI mirror including rebase and workspace sync checks
make pre-merge
```

### Git Hooks

The repo ships pre-commit and pre-push hooks in `.githooks/`. Each hook dispatches to the corresponding Makefile gate:

| Hook | Gate | Skips |
|------|------|-------|
| `pre-commit` | `make pre-commit` | `make hooks-skip-pre-commit` |
| `pre-push` | `make pre-merge` | `make hooks-skip-pre-merge` |

**Host/bare-metal:** the hook dispatches via `-ctzd` (the gate runs inside the tools container). Activate with `make hooks-install` once per clone.

**ECP containers:** the dev container entrypoint sets `core.hooksPath` automatically on every start, covering both the SSH dev container (`make ctzdev-start`) and the VS Code/Codium devcontainer. The `ECP_CONTAINER=1` env var baked into the builder image (inherited by tools and dev) tells the hook to run `make <target>` directly instead.

```bash
# One-time per-clone activation on the host
make hooks-install

# Persistently disable a hook
make hooks-skip-pre-commit      # re-enable: make hooks-unskip-pre-commit
make hooks-skip-pre-merge       # re-enable: make hooks-unskip-pre-merge

# Skip a single commit or push without disabling the hook
git commit --no-verify
git push --no-verify
```

### Containerized Development (Ephemeral)

Append `-ctzd` to any root Makefile target to run it inside the tools container. The tools image is built automatically on first use.

```bash
make test-ctzd                                  # all tests, inside container
make foundation/gateway-lint-ctzd               # lint one module
make generate-api-ctzd                          # codegen inside container
make pre-commit-ctzd                            # full local gate, containerized
make test-ctzd RUN=TestCreateWorkspace          # pass variables through
```

The `-ctzd` wrapper (`%-ctzd` in `.common.mk`) mounts the repo root, the container socket, and the `.cache/` directory into the tools container. Go module and build caches persist across runs via `.cache/go` and `.cache/go-build`.

Variables forwarded into the container: `RUN`, `PKG`, `RELPATH`, `GH_TOKEN`, `BASE_REF`.

### Persistent Dev Container

For an interactive, full-featured shell with neovim and gopls:

```bash
# Start the dev container (SSH on port 2222)
make ctzdev-start

# Connect via SSH
ssh -p 2222 dev@localhost          # Docker
ssh -p 2222 $USER@localhost        # Podman (preserves host username)

# Check status
make ctzdev-status

# Stop and remove
make ctzdev-stop
```

The dev container mounts the repo root and the host SSH keys, and provides Docker-in-Docker via socket mount so KIND clusters can be created from inside the container. The `HOST_WORKSPACE` and `HOST_SOCKET` environment variables are set so nested `docker`/`kind` calls reference the correct host paths.

### VS Code / Codium Dev Container

The `.devcontainer/` directory provides a pre-configured VS Code/Codium devcontainer:

- `devcontainer.json` — declares the compose stack and the `golang.go` extension.
- `compose.yml` — static service definition (host networking, workspace bind mount).
- `compose.override.yml` and `.env` — generated at startup time by `initializeCommand`, which runs `ci/scripts/devcontainer-init.sh` on the host to detect the container backend and write backend-specific flags.

Open the repository in VS Code/Codium and select **Reopen in Container**. The tools image is built automatically if not already present.

## Container Image Chain

The 3 images form a layered chain. Each layer adds tooling on top of the previous one.

### Builder Image (`ci/container/builder/`)

| Attribute | Value |
|-----------|-------|
| Base | `golang:1.26.4-trixie` |
| Contains | Go toolchain, all codegen/lint/security tools (pinned versions) |
| Published by | CI (`builder-publish.yaml`) to `ghcr.io/eu-sovereign-cloud/ecp-builder` |
| Pinned at | `.builder-digest` (committed to git) |

The builder image is the foundation for all CI jobs. It is rebuilt by CI whenever `ci/container/builder/`, `ci/tools/`, `ci/scripts/`, `.config.mk`, `.common.mk`, `Makefile`, or `builder-build-push` / `builder-pr-publish` action files change, or when `builder-publish.yaml` or `pre-merge.yaml` themselves change. The full path set is owned by `ci/scripts/paths-filter-gen.sh`.

When a branch modifies any of those inputs, the Makefile automatically detects the stale `.builder-digest` and builds/uses a local image — no `BUILDER_SOURCE=local` is required. `%-ctzd` targets and `ctzdev-start` just work. To force a rebuild from scratch:

```bash
make builder-rebuild   # --no-cache rebuild of the builder
make tools-build       # propagate downstream (auto on next -ctzd)
```

### Tools Image (`ci/container/tools/`)

| Attribute | Value |
|-----------|-------|
| Base | `builder` |
| Adds | Docker CLI (static binary), KIND, kubectl, GitHub CLI, bash completion, coloring |
| Tag | `localhost/ecp/tools:<version>-trixie-go-v1.26.4` |
| Built by | `make tools-build` (auto-triggered by `-ctzd` targets if missing) |

This image is what the `-ctzd` targets and the devcontainer use.

### Dev Image (`ci/container/dev/`)

| Attribute | Value |
|-----------|-------|
| Base | `tools` |
| Adds | OpenSSH server, neovim, gopls, sudo |
| Tag | `localhost/ecp/dev:<version>-trixie-go-v1.26.4` |
| Built by | `make dev-build` (auto-triggered by `ctzdev-start` if missing) |

### Runner Image (`ci/container/runner/`)

Minimal distroless base (`gcr.io/distroless/static-debian13`) for production deployments of ECP components.

### Image Build Targets

| Target | Description |
|--------|-------------|
| `make tools-build` | Build tools image (pulls builder from registry) |
| `make dev-build` | Build dev image |
| `make images-build` | Build all 3 images from local sources |
| `make builder-rebuild` | Force-rebuild builder image (no Docker cache) |
| `make tools-rebuild` | Force-rebuild tools image (no Docker cache) |
| `make images-rebuild` | Force-rebuild all 3 images |
| `make images-clean` | Remove all 3 local images |

## Makefile Architecture

### Configuration Files

| File | Purpose |
|------|---------|
| `.config.mk` | Version pins (Go, tools, gopls, dlv, KIND, kubectl, gh CLI) and container registry coordinates |
| `.common.mk` | Container backend detection, image build/ensure targets, `-ctzd` wrapper, persistent dev container targets |
| `ci/tools/tools.mk` | `tools-install` target: installs Go dev tools to `ci/tools/bin/` |

### Root Makefile Target Reference

| Category | Target(s) | Description |
|----------|-----------|-------------|
| **Verification** | `test`, `<module>-test` | Unit tests with race detector (`-race`). Optional `RUN=<regex>` filter. |
| | `lint`, `<module>-lint` | golangci-lint with `.golangci.yml` config |
| | `gofmt`, `<module>-gofmt` | Auto-fix formatting via `golangci-lint fmt` |
| | `gofmt-check`, `<module>-gofmt-check` | Format check only (non-zero exit on diff; used by CI) |
| | `vuln`, `<module>-vuln` | govulncheck vulnerability scan (single-module mode) |
| | `gosec`, `<module>-gosec` | gosec security scan |
| **Code Generation** | `generate-api` | Generate CRDs + typed models from OpenAPI spec |
| | `generate-api-verify` | Same, but fails if git tree is dirty afterward (CI gate) |
| **Dependency Mgmt** | `tidy`, `<module>-tidy` | `go mod tidy` per module |
| | `go-get`, `<module>-go-get PKG=<pkg>` | `go get <pkg>` per module + tidy |
| | `workspace-sync` | `go work sync` |
| | `workspace-verify` | `workspace-sync` + git-cleanliness gate (CI gate) |
| **CI Gates** | `pre-commit` | `go-sdk-verify generate-api-verify test lint gofmt-check vuln gosec` |
| | `pre-merge` | Same, plus `gh-token-ensure branch-rebase-verify workspace-verify` |
| | `branch-rebase-verify` | Verify current branch is rebased onto its PR target |
| **Container Images** | `tools-build`, `dev-build`, `images-build` | Build image(s) |
| | `builder-rebuild`, `tools-rebuild`, `images-rebuild` | Force-rebuild (bypass cache) |
| | `images-clean` | Remove local images |
| **Dev Container** | `ctzdev-start` | Start persistent dev container (SSH on port 2222) |
| | `ctzdev-stop` | Stop and remove dev container |
| | `ctzdev-status` | Check dev container status |
| **Workspace** | `workspace-use-add RELPATH=<path>` | Add module to `go.work` |
| | `workspace-use-drop RELPATH=<path>` | Remove module from `go.work` |
| **Utilities** | `tools-install` | Install pinned Go dev tools to `ci/tools/bin/` |
| | `submodules` | Sync and update git submodules |
| | `hooks-install` | Activate git hooks (`core.hooksPath = .githooks`) — one-time per clone |
| | `hooks-skip-pre-commit` | Disable pre-commit hook persistently (`hooks.skipPreCommit = true`) |
| | `hooks-unskip-pre-commit` | Re-enable pre-commit hook |
| | `hooks-skip-pre-merge` | Disable pre-push hook persistently (`hooks.skipPreMerge = true`) |
| | `hooks-unskip-pre-merge` | Re-enable pre-push hook |
| | `gh-token-ensure` | Validate or refresh the cached GitHub CLI token |
| | `print-<VAR>` | Print any computed Make variable (e.g., `make -s print-TOOLS_IMAGE`) |
| | `sh` | Open a bash shell |

### The `-ctzd` Pattern

Any target `FOO` defined at the root can be run as `FOO-ctzd`. The wrapper:

1. Ensures the tools image exists (`_tools-ensure-image`), building it if needed.
2. Runs `docker run --rm -it <flags> $(TOOLS_IMAGE) make FOO`.
3. Mounts the repo root, container socket, and caches.
4. Forwards variables: `RUN`, `PKG`, `RELPATH`, `GH_TOKEN`, `BASE_REF`.

`%-container` is an alias for `%-ctzd`.

### Sub-Module Makefiles

| Makefile | Key Targets |
|----------|-------------|
| `foundation/gateway/Makefile` | `run-global-server`, `run-regional-server`, `build-gateway`, `create-dev-clusters`, `clean-dev-clusters` |
| `foundation/persistence/Makefile` | `generate-all`, `generate-models`, `generate-crds`, `clean-generated`, `clean-crds` |
| `csp/dummy/Makefile` | `build`, `deploy`, `kind-start`, `kind-stop`, `test-integration` |
| `csp/e2e/Makefile` | `build-all`, `push-all`, `deploy-all`, `kind-start`, `kind-stop`, `kind-load-all`, `test-all` |
| `csp/ionos/deploy/Makefile` | `install-crossplane`, `install-provider`, `install-all`, `install-on-regional` |
| `csp/ionos_e2e/Makefile` | `secatest-scaffolding`, `secatest`, `secatest-all`, `secatest-clean` |

## CI Pipeline (GitHub Actions)

### `pre-merge.yaml` — PR validation

Triggered on `pull_request` to `main` (`opened`, `synchronize`, `reopened`, `closed`).

```
Stage 1 — cheap gates, run in parallel
  pr-title         Validate PR title (conventional commits via amannn/action-semantic-pull-request)
  module-diff      Detect which Go modules changed (dorny/paths-filter, config derived from go.work)
  branch-rebase    Verify branch is rebased onto its target (make branch-rebase-verify)

Stage 2 — depends on Stage 1
  builder-publish-pr   Ensure a builder image exists for this PR:
                         - If no builder inputs changed → alias :main as :pr-<N>
                         - If inputs changed → full rebuild, push as :pr-<N>

Stage 3 — parallel, per changed module, inside the builder container
  workspace-verify     make workspace-verify
  generate-api         make generate-api-verify
  test                 make <module>-test        (matrix over changed modules)
  lint                 make <module>-lint         (matrix)
  gofmt                make <module>-gofmt-check  (matrix)
  vuln                 make <module>-vuln         (matrix)
  gosec                make <module>-gosec        (matrix)

Cleanup — on PR close
  builder-cleanup   Delete :pr-<N> and :pr-<N>-buildcache tags from GHCR
```

**Checkout strategy:** all jobs pin checkout to `github.event.pull_request.head.sha` (not the synthetic merge commit). This means CI validates exactly the tree the contributor sees locally, avoiding surprise failures caused by regressions on `main`.

**Module filtering:** Stage 3 jobs only run for modules that have changed files. `module-diff` generates a `paths-filter` config from `go.work` at runtime (`make -s print-paths-filter` → `ci/scripts/paths-filter-gen.sh`), so the filter stays in sync with the workspace automatically — adding a new module to `go.work` is all that's needed. When builder inputs change, `module-matrix.sh` expands the matrix to every module so all checks run against the rebuilt toolchain.

### `builder-publish.yaml` — Builder image publishing

Triggers on every push to `main` (and `workflow_dispatch`). A `detect` job runs `make -s print-paths-filter` and uses `dorny/paths-filter` against `github.event.before` to check whether builder inputs changed; the `publish` job is skipped when they have not. The path set is owned by `ci/scripts/paths-filter-gen.sh` — the single source of truth shared with `pre-merge.yaml` and `builder-pr-publish`.

1. Builds and pushes the builder image to `ghcr.io/eu-sovereign-cloud/ecp-builder` with tags `:main` and `:sha-<12-char-sha>`.
2. Uses registry-based BuildKit cache for fast incremental rebuilds.
3. Opens an automated PR to bump `.builder-digest` on `main` and immediately **merges it** (squash) in the same workflow run. No human action is needed — the merge commit on `main` is the moment developers and CI adopt the new builder.

Runs are serialized (`cancel-in-progress: false`) so the second run benefits from the first's registry cache.

**One-time repo setup required:**
- **Package visibility:** after the first push, set the GHCR package to Public at `https://github.com/orgs/eu-sovereign-cloud/packages/container/ecp-builder/settings`.
- **PR permission:** in **Settings → Actions → General → Workflow permissions**, enable *"Allow GitHub Actions to create and approve pull requests"*. Without this the digest-bump step fails with `"GitHub Actions is not permitted to create or approve pull requests"`. Because the bump PR is opened by the default `GITHUB_TOKEN`, `pre-merge.yaml` will not be triggered on it — this is intentional since the publish job already validated the image build.

### Adding a New Go Module to CI

1. Create the module directory with a `go.mod` file.
2. Add it to the workspace: `make workspace-use-add RELPATH=<path/to/module>`
3. If the module imports other workspace members, add a `replace` directive:
   `go work edit -replace <modpath>=./<path/to/module>`
4. Run `make workspace-sync` to update `go.work.sum`.
5. Commit `go.work` and `go.work.sum`.

CI picks up the new module automatically: `print-paths-filter` regenerates the `dorny/paths-filter` configuration from `go.work` at run time.

## `ci/scripts/` Reference

| Script | Purpose |
|--------|---------|
| `branch-rebase-verify.sh` | Verify current branch is rebased onto its PR target |
| `container-image-exists.sh` | Check whether a container image exists locally |
| `container-runtime-detect.sh` | Detect Docker vs Podman backend |
| `container-security-opts.sh` | Emit `--security-opt` flags appropriate for the backend |
| `container-socket-path.sh` | Resolve the container socket path for the backend |
| `container-user-flags.sh` | Emit `--user` / `--userns` flags for the backend |
| `container-volume-opts.sh` | Emit SELinux volume label suffix (`:Z` or empty) |
| `devcontainer-init.sh` | Generate `.devcontainer/.env` and `compose.override.yml` for the VS Code devcontainer |
| `gh-token-ensure.sh` | Validate or re-authenticate the cached GitHub CLI token |
| `git-hook-run.sh` | Shared dispatcher for `.githooks/` — checks skip config, detects container context, runs `make <target>[-ctzd]` |
| `git-tree-clean-verify.sh` | Fail if the git working tree is dirty (used by verify targets) |
| `go-modules.sh` | Print CI-relevant Go modules from `go.work` (single source of truth for `GO_MODULES` and `paths-filter-gen.sh`) |
| `gofmt-check.sh` | Run `golangci-lint fmt --diff` and fail on any diff |
| `image-inputs-changed.sh` | Detect whether a container image's build inputs differ from the `.builder-digest` commit (local stale-image detection) |
| `kind-cgroup-preflight.sh` | Check KIND cgroup delegation prerequisites for rootless Podman |
| `module-matrix.sh` | Resolve the pre-merge CI module check-matrix: full set when builder inputs changed, otherwise the changed-module set |
| `paths-filter-gen.sh` | Emit the `dorny/paths-filter` YAML config: a `builder` block plus one block per CI-relevant Go module |
| `tool-ensure-go.sh` | Ensure a Go tool binary is present (used by `tools-install`) |
| `verify-run.sh` | Wrap a command with a pass/fail header for consistent CI output |
