#!/usr/bin/env bash
# Ensure a valid GitHub CLI token is cached.
#
# Usage: gh-token-ensure.sh <token-file>
#   <token-file>  path where the token is persisted (e.g. .cache/gh-token)
#
# Reads GH_TOKEN from the environment. If the token is valid (verified via
# `gh api /user`) the script is a no-op. Otherwise it triggers an interactive
# `gh auth login`, captures the fresh token, and writes it to <token-file>.
#
# On success the valid token is printed to stdout so the caller can capture it.
#
# Exit code: 0 on success, 1 on failure

set -euo pipefail

token_file="${1:?Usage: gh-token-ensure.sh <token-file>}"

# Check if the current token is valid
if [ -n "${GH_TOKEN:-}" ] && GH_TOKEN="${GH_TOKEN}" gh api /user >/dev/null 2>&1; then
  echo "==> gh-token: valid" >&2
  printf '%s' "${GH_TOKEN}"
  exit 0
fi

echo "==> gh-token: no valid token — starting authentication" >&2

gh auth login

token=$(gh auth token) || {
  echo "::error::gh auth token failed" >&2
  exit 1
}

mkdir -p "$(dirname "${token_file}")"
printf '%s' "${token}" > "${token_file}"
echo "==> gh-token: saved to ${token_file}" >&2

printf '%s' "${token}"
