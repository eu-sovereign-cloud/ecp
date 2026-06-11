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

## Adding a New Go Module

1. Create the directory and initialize `go.mod`.
2. Add to the workspace: `make workspace-use-add RELPATH=<path>`
3. If the module imports other workspace members (e.g., `foundation/gateway`), add a `replace` directive:
   `go work edit -replace <modpath>=./<path>`
4. Sync: `make workspace-sync`
5. Commit `go.work` and `go.work.sum`.

CI auto-discovers the module via `print-paths-filter` — no workflow changes are needed.

To exclude a module from standard product CI checks (e.g., test harnesses, tool modules), add it to `GO_MODULES_EXCLUDE` in `.common.mk`.

## Generated Code

Several files are generated and must not be edited by hand:

- `foundation/persistence/generated/types/zz_generated_*.go` — Go types from OpenAPI spec
- `foundation/persistence/generated/crds/*.yaml` — CRD YAML from controller-gen

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
