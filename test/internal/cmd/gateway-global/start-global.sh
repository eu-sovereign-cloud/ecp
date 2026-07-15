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
#   AUTH_PLUGIN=jwt             — verify standard signed JWTs instead of dummy tokens
#   AUTHZ_ENABLED=false         — authn-only (identity checked, no RBAC)
#   AUTHZ_IMPL=direct           — use per-request RBAC checker instead of cached
#   DUMMY_AUTH_USERS=/path      — path to username→password JSON (default /app/users.json)
#   JWT_SIGNING_METHOD=...      — expected JWT alg when AUTH_PLUGIN=jwt (default ES256)
#   JWT_SECRET=/path            — key file when AUTH_PLUGIN=jwt: the raw HMAC secret for
#                                 HS*, a PEM public key otherwise (default /app/jwt.pub)
#   AUTHZ_SKIP_PROVIDERS=...    — comma-separated provider IDs served authn-only
#                                 (binary default: seca.region — the tenant-less region catalog)
: "${AUTH_ENABLED:=true}"
: "${AUTH_PLUGIN:=dummy}"
: "${AUTHZ_ENABLED:=true}"
: "${AUTHZ_IMPL:=cached}"
: "${DUMMY_AUTH_USERS:=/app/users.json}"
: "${JWT_SIGNING_METHOD:=ES256}"
: "${JWT_SECRET:=/app/jwt.pub}"

AUTH_FLAGS=""
if [ "$AUTH_ENABLED" = "true" ]; then
  if [ "$AUTH_PLUGIN" = "jwt" ]; then
    AUTH_FLAGS="--auth-enabled --auth-plugin=jwt --jwt-signing-method=$JWT_SIGNING_METHOD --jwt-secret=$JWT_SECRET"
  else
    AUTH_FLAGS="--auth-enabled --auth-plugin=dummy --dummy-auth-users=$DUMMY_AUTH_USERS"
  fi
  if [ "$AUTHZ_ENABLED" = "true" ]; then
    AUTH_FLAGS="$AUTH_FLAGS --authz-enabled"
    [ "$AUTHZ_IMPL" = "cached" ] && AUTH_FLAGS="$AUTH_FLAGS --authz-cache"
    [ -n "$AUTHZ_SKIP_PROVIDERS" ] && AUTH_FLAGS="$AUTH_FLAGS --authz-skip-providers=$AUTHZ_SKIP_PROVIDERS"
  else
    AUTH_FLAGS="$AUTH_FLAGS --authz-enabled=false"
  fi
fi

# The 'ecp' binary runs the global server by default.
# We pass any extra arguments ($@) to the binary.
echo "Starting global gateway..."
# shellcheck disable=SC2086
./ecp globalapiserver --host="$GLOBAL_HOST" --port="$GLOBAL_PORT" $KUBECONFIG_FLAG $AUTH_FLAGS "$@"
