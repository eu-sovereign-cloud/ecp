# Contributing to ECP

## Branch Model

- `main` is the integration branch — all work merges here.
- Feature branches follow the pattern `<type>/<short-description>` (e.g., `feat/block-storage-resize`, `ci/improve-caching`).
- All branches must be rebased onto `main` before merging. This is enforced by CI (`branch-rebase-verify`) and by the `make pre-merge` gate locally.

## PR Conventions

PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | When to use |
|------|-------------|
| `feat` | New feature or behavior |
| `fix` | Bug fix |
| `refactor` | Code change with no behavior change |
| `test` | Adding or updating tests |
| `docs` | Documentation changes only |
| `ci` | CI or build system changes |
| `chore` | Dependency bumps, generated file updates |

This is enforced by the `pr-title` CI job (`amannn/action-semantic-pull-request`).

CI runs per-module: only modules with changed files are tested and linted. Unrelated modules are not affected by your PR.

## Local Validation

Before committing:
```bash
make pre-commit          # generate-api-verify + test + lint + gofmt-check + vuln + gosec
make pre-commit-ctzd     # same, inside the tools container
```

Before pushing / opening a PR:
```bash
make pre-merge           # same as pre-commit, plus branch-rebase-verify + workspace-verify
make pre-merge-ctzd      # same, inside the tools container
```

`pre-merge` requires a valid GitHub CLI token to discover the PR target branch. Run `make gh-token-ensure` once to cache it.

## Code Style

- **Linting:** `golangci-lint` with the configuration in `.golangci.yml`.
- **Formatting:** `gofumpt` (applied via `golangci-lint fmt`). Run `make gofmt` to fix formatting in place; `make gofmt-check` to check without modifying (what CI runs).
- Keep `make lint` and `make gofmt` green before pushing.

## Import Alias Convention

All cross-module imports follow the canonical `<resource><layer>` alias convention. Importas lint enforcement is **planned** (the `alias:` list in `.golangci.yml` is currently empty; aliases are followed by hand convention). Examples:

```go
import (
    bsdom  "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
    bsk8s  "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/backend/kubernetes"
    bsrest "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/frontend/rest"
    netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1"
)
```

The alias convention keeps deep slice paths out of the readable code. See [CODEGEN.md](CODEGEN.md) for the full alias table.

## Module Boundaries

The repo enforces a strict import DAG. Violations fail both `golangci-lint depguard` and the `framework-isolation` CI lane:

```
framework/kernel      ← pure leaf (no other framework/* imports)
framework/persistence → kernel
framework/backend     → persistence, kernel
framework/frontend    → kernel
(any framework/*)     ↛ resources   ← COMPILER-ENFORCED (separate Go modules)
```

**Rules:**
- No `framework/*` package may import `resources`. This is a Go build error under `GOWORK=off` and is caught by the `framework-isolation` CI lane.
- Within `framework/`, layers may only import packages at the same or lower level in the DAG above.
- `resources` may freely import `framework`. Nothing except `gateway`, `csp/*`, and `test/e2e` may import `resources`.

## Adding a New Go Module

1. Create the directory and initialize `go.mod`.
2. Add to the workspace: `make workspace-use-add RELPATH=<path>`
3. If the module imports other workspace members, add `require` + `replace` directives in `go.mod`:
   ```
   require github.com/eu-sovereign-cloud/ecp/framework v0.0.1
   replace github.com/eu-sovereign-cloud/ecp/framework => ../../framework
   ```
4. Sync: `make workspace-sync`
5. Commit `go.work` and `go.work.sum`.

CI auto-discovers the module via `print-paths-filter` — no workflow changes are needed.

To exclude a module from standard product CI checks (e.g., test harnesses, tool modules), add it to `GO_MODULES_EXCLUDE` in `.common.mk`.

## Adding a New Resource Slice

1. Create the slice directory: `resources/{global,regional}/<group>/<resource>/vN/`
2. Add `domain.go` (`package v1`) with the canonical domain type and identity consts (`Kind`, `Resource`, `Group`, `Version`, and a provider identifier).
3. Add `backend/kubernetes/` with CR types, GVR, adapters, controller, plugin interface, and plugin handler.
4. Add `frontend/rest/` with converter and HTTP handlers implementing the go-sdk `ServerInterface`.
5. Run `make generate-api` to route generated types into the new slice.
6. Register the handler in `gateway/cmd/` and the controller in the relevant CSP `cmd/main.go`.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the full per-slice hexagon description and [PLUGINS.md](PLUGINS.md) for the builder-inversion wiring pattern.

## Generated Code

Several files are generated and must not be edited by hand:

- `resources/{scope}/{group}/{resource}/vN/backend/kubernetes/zz_generated_schema.go` — Go types from go-sdk schema (per-slice; run `go generate ./...` in `resources/`)
- `framework/persistence/kubernetes/schema/v1/` — shared CR envelope types (run `make generate-api`)
- `framework/persistence/kubernetes/crds/*.yaml` — CRD YAML from controller-gen (**planned**; `generate-crds` target is scaffolded but no sources emit yet)
- `**/zz_generated.deepcopy.go`, `**/zz_generated.conditions.go` — controller-gen and conditioned-gen output

After changing the OpenAPI specs in `modules/go-sdk`, regenerate:
```bash
make generate-api
```

CI runs `make generate-api-verify` in every PR. It fails if the committed generated files differ from a fresh generation run.

See [CODEGEN.md](CODEGEN.md) for details.

## Alpha-Phase Considerations

ECP is at version `v0.1.0-alpha1-preview`. During this phase:

- **Breaking changes are expected** across API surface, CRD schemas, and Go module interfaces.
- Deprecation notices are not guaranteed before a breaking change.
- Document any breaking changes prominently in the PR description so reviewers and downstream consumers are aware.
- The alpha status will be revised when the API surface and CRD schemas stabilize toward a v1.0 release.
