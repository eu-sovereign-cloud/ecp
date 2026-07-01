#!/bin/bash
set -e

# This script starts the global gateway server.
# It uses environment variables for configuration and passes them as flags to the 'ecp' binary.

# Set default values if environment variables are not provided.
: "${GLOBAL_HOST:=0.0.0.0}"
: "${GLOBAL_PORT:=8080}"

# The kubeconfig default is handled by the binary itself, but we can pass it if set.
KUBECONFIG_FLAG=""
if [ -n "$KUBECONFIG" ]; then
  KUBECONFIG_FLAG="--kubeconfig=$KUBECONFIG"
fi

# Auth defaults: dummy authn + cached authz enabled.
# Override via env vars before calling this script:
#   AUTH_ENABLED=false          — no auth (unauthenticated mode)
#   AUTHZ_ENABLED=false         — authn-only (identity checked, no RBAC)
#   AUTHZ_IMPL=direct           — use per-request RBAC checker instead of cached
#   DUMMY_AUTH_USERS=/path      — path to username→password JSON (default /app/users.json)
: "${AUTH_ENABLED:=true}"
: "${AUTHZ_ENABLED:=true}"
: "${AUTHZ_IMPL:=cached}"
: "${DUMMY_AUTH_USERS:=/app/users.json}"

AUTH_FLAGS=""
if [ "$AUTH_ENABLED" = "true" ]; then
  AUTH_FLAGS="--auth-enabled --dummy-auth-users=$DUMMY_AUTH_USERS"
  if [ "$AUTHZ_ENABLED" = "true" ]; then
    AUTH_FLAGS="$AUTH_FLAGS --authz-enabled"
    [ "$AUTHZ_IMPL" = "cached" ] && AUTH_FLAGS="$AUTH_FLAGS --authz-cache"
  else
    AUTH_FLAGS="$AUTH_FLAGS --authz-enabled=false"
  fi
fi

# The 'ecp' binary runs the global server by default.
# We pass any extra arguments ($@) to the binary.
echo "Starting global gateway..."
# shellcheck disable=SC2086
./ecp globalapiserver --host="$GLOBAL_HOST" --port="$GLOBAL_PORT" $KUBECONFIG_FLAG $AUTH_FLAGS "$@"
