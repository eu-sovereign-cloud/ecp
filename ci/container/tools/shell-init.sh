# Skip if not running under bash (e.g. dash via /bin/sh)
[ -z "${BASH_VERSION:-}" ] && return 0 2>/dev/null || true

# Git branch for prompt
__git_ps1_branch() {
  local branch
  branch=$(git symbolic-ref --short HEAD 2>/dev/null || git rev-parse --short HEAD 2>/dev/null)
  [ -n "${branch}" ] && echo " (${branch})"
}

# Prompt coloring with git branch
export PS1="\[\033[1;34m\][ecp-tools]\[\033[0m\] \[\033[1;32m\]\w\[\033[0m\]\[\033[1;33m\]\$(__git_ps1_branch)\[\033[0m\] \$ "

# Enable color output
alias ls="ls --color=auto"
alias grep="grep --color=auto"

# Enable bash completion
if [ -f /etc/bash_completion ]; then
  . /etc/bash_completion
fi
