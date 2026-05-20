#!/usr/bin/env bash
# Verify an action produced no unaccounted-for changes.
#
# Usage: git-tree-clean-verify.sh <mode> <repo-root> <action> <remediation> [path...]
#   <mode>          --against-index : compare the working tree to the staging
#                                     area. Staged output is accepted; only
#                                     unstaged or untracked changes the action
#                                     produced fail. Use for pre-commit gates.
#                   --against-head  : the scoped tree must match HEAD — nothing
#                                     uncommitted at all. Use for pre-merge (CI)
#                                     gates.
#   <repo-root>     absolute path to the repository root
#   <action>        human-readable name of the action that was just run
#                   (e.g. "workspace-sync", "generate-api")
#   <remediation>   command the developer should run to fix the issue
#                   (e.g. "make workspace-sync", "make generate-api")
#   [path...]       optional paths to scope the check (relative to repo root).
#                   When omitted the entire tree is checked.
#
# Exit code: 0 if clean, 1 if dirty, 2 on usage error

set -euo pipefail

mode="${1:?Usage: git-tree-clean-verify.sh <mode> <repo-root> <action> <remediation> [path...]}"
repo_root="${2:?Usage: git-tree-clean-verify.sh <mode> <repo-root> <action> <remediation> [path...]}"
action="${3:?Usage: git-tree-clean-verify.sh <mode> <repo-root> <action> <remediation> [path...]}"
remediation="${4:?Usage: git-tree-clean-verify.sh <mode> <repo-root> <action> <remediation> [path...]}"
shift 4

cd "${repo_root}"

case "${mode}" in
  --against-index)
    # Unstaged modifications to tracked files + new untracked files the
    # action produced. Staged content is intentionally accepted — a pre-commit
    # gate validates what is about to be committed, not what is already there.
    dirty=$(
      git diff --name-only -- "$@"
      git ls-files --others --exclude-standard -- "$@"
    )
    hint="stage the results"
    ;;
  --against-head)
    # The scoped tree must match HEAD — nothing uncommitted at all.
    dirty=$(git status --porcelain -- "$@" | awk '{ print $2 }')
    hint="commit the results"
    ;;
  *)
    echo "error: unknown mode '${mode}' (expected --against-index or --against-head)" >&2
    exit 2
    ;;
esac

if [ -n "${dirty}" ]; then
  echo "::error::${action} produced changes in:"
  echo "${dirty}" | sed 's/^/  /'
  echo ""
  echo "  Run '${remediation}' and ${hint}."
  exit 1
fi
