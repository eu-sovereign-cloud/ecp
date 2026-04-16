#!/usr/bin/env bash
# Run a verify command and print a standardized result box.
#
# Usage: verify-run.sh <target> <description> -- <command> [args...]
#   <target>       make target name (e.g. "branch-rebase-verify")
#   <description>  what is being verified (e.g. "Branch is rebased onto target")
#
# The command's stdout/stderr flow to the terminal normally. After it
# completes, a colored result box is printed:
#   - Success (exit 0): green box to stdout
#   - Failure (exit >0): red box to stderr, exits with the original code
#
# Exit code: same as the wrapped command

set -uo pipefail

target="${1:?Usage: verify-run.sh <target> <description> -- <command> [args...]}"
description="${2:?Usage: verify-run.sh <target> <description> -- <command> [args...]}"
shift 2

# Consume the "--" separator
if [ "${1:-}" = "--" ]; then
  shift
fi

# Run the command
rc=0
"$@" || rc=$?

# Box dimensions
WIDTH=80
INNER=$((WIDTH - 4))

# Color palette
RST="\033[0m"
if [ "${rc}" -eq 0 ]; then
  BC="\033[32m"       # dark green border
  TC="\033[1;32m"     # bold green text
  RESULT="SUCCEED"
  FD=1
else
  BC="\033[31m"       # dark red border
  TC="\033[1;31m"     # bold red text
  RESULT="FAILED"
  FD=2
fi

# Print the box
_border=$(printf '%0.s#' $(seq 1 ${WIDTH}))
_title="${target}: ${description}"
_result="Result: ${RESULT}"

{
  echo ""
  printf "${BC}%s${RST}\n" "${_border}"
  printf "${BC}#%*s#${RST}\n" $((WIDTH - 2)) ""
  printf "${BC}# ${TC}%-${INNER}s${BC} #${RST}\n" "${_title}"
  printf "${BC}#%*s#${RST}\n" $((WIDTH - 2)) ""
  printf "${BC}# ${TC}%-${INNER}s${BC} #${RST}\n" "${_result}"
  printf "${BC}#%*s#${RST}\n" $((WIDTH - 2)) ""
  printf "${BC}%s${RST}\n" "${_border}"
  echo ""
} >&${FD}

exit ${rc}
