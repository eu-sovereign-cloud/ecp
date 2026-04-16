#!/usr/bin/env bash
# Verify the current branch is rebased onto its PR target.
#
# Usage: branch-rebase-verify.sh <repo-root>
#
# Resolves the base branch in order:
#   1. $BASE_REF environment variable (set by CI)
#   2. `gh pr view` (open PR for current HEAD)
#   3. `gh repo view` (repo default branch — fallback)
#
# When GH_TOKEN is set, git fetch uses HTTPS with token auth instead of SSH.
# This makes the script work inside containers that lack SSH keys.
#
# Exit code: 0 if rebased, 1 if not, 2 if base branch cannot be resolved

set -euo pipefail

repo_root="${1:?Usage: branch-rebase-verify.sh <repo-root>}"

# Ensure SSH always writes known_hosts inside $HOME/.ssh/ so it never lands
# in the repository working tree (which is the CWD inside the container).
mkdir -p "${HOME}/.ssh"
export GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o UserKnownHostsFile=${HOME}/.ssh/known_hosts}"

# Build extra git flags: if GH_TOKEN is available, rewrite git@github.com:
# URLs to HTTPS so the fetch works without SSH keys (e.g. inside containers).
_git_auth_flags=()
if [ -n "${GH_TOKEN:-}" ]; then
  _git_auth_flags+=(-c "url.https://oauth2:${GH_TOKEN}@github.com/.insteadOf=git@github.com:")
fi

base_ref="${BASE_REF:-}"

if [ -z "${base_ref}" ]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "::error::gh CLI not found; install it or run with BASE_REF=<branch>"
    exit 2
  fi

  base_ref=$(gh pr view --json baseRefName -q .baseRefName 2>/dev/null) ||
  base_ref=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null) || {
    echo "::error::could not resolve target branch via gh; set BASE_REF=<branch>"
    exit 2
  }
fi

echo "==> branch-rebase-verify: base=origin/${base_ref}"

git -C "${repo_root}" "${_git_auth_flags[@]}" fetch --quiet origin "${base_ref}"

base_tip=$(git -C "${repo_root}" rev-parse "origin/${base_ref}")
merge_base=$(git -C "${repo_root}" merge-base HEAD "origin/${base_ref}")

if [ "${merge_base}" != "${base_tip}" ]; then
  echo "::error::branch is not rebased onto origin/${base_ref}"
  echo "  origin/${base_ref} tip : ${base_tip}"
  echo "  merge-base with HEAD  : ${merge_base}"
  echo "  fix: git fetch origin ${base_ref} && git rebase origin/${base_ref}"
  exit 1
fi

echo "OK: HEAD is rebased onto origin/${base_ref}"
