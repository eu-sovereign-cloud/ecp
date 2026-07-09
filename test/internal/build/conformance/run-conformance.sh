#!/bin/bash
# Run the SECA conformance suite (secatest) from inside the conformance image.
#
# Configuration is taken from environment variables set on the Deployment; the
# optional first argument overrides the scenarios filter. This runner is generic:
# it targets whatever gateway PROVIDER_REGION_V1 points at, so the delegator
# plugin behind that gateway (dummy, aruba, ionos, ...) is what gets exercised.
set -eo pipefail

SCENARIO="${1:-$SCENARIOS_FILTER}"

# secatest groups [provider.region.v1 client.auth.token client.region client.tenant
# report.results.path] as all-or-none, so report.results.path is mandatory whenever
# we set the others. Default to a writable path in the pod (the image runs as a
# non-root user, so /test is read-only); the deployment can override via RESULTS_PATH.
RESULTS_PATH="${RESULTS_PATH:-/tmp/conformance-results}"

args=(
    run
    --scenarios.filter="$SCENARIO"
    --provider.region.v1="$PROVIDER_REGION_V1"
    --client.auth.token="$CLIENT_AUTH_TOKEN"
    --client.region="$CLIENT_REGION"
    --client.tenant="$CLIENT_TENANT"
    --report.results.path="$RESULTS_PATH"
)

# Optional settings — appended only when provided, so simple scenarios stay simple
# while lifecycle scenarios can supply the extra inputs they need.
[[ -n "$PROVIDER_AUTHORIZATION_V1" ]] && args+=(--provider.authorization.v1="$PROVIDER_AUTHORIZATION_V1")
[[ -n "$SCENARIOS_USERS" ]]          && args+=(--scenarios.users="$SCENARIOS_USERS")
[[ -n "$SCENARIOS_CIDR" ]]           && args+=(--scenarios.cidr="$SCENARIOS_CIDR")
[[ -n "$SCENARIOS_PUBLIC_IPS" ]]     && args+=(--scenarios.public.ips="$SCENARIOS_PUBLIC_IPS")
[[ -n "$RETRY_BASE_DELAY" ]]         && args+=(--retry.base.delay="$RETRY_BASE_DELAY")
[[ -n "$RETRY_BASE_INTERVAL" ]]      && args+=(--retry.base.interval="$RETRY_BASE_INTERVAL")
[[ -n "$RETRY_MAX_ATTEMPTS" ]]       && args+=(--retry.max.attempts="$RETRY_MAX_ATTEMPTS")

echo "Running: secatest ${args[*]}"
./secatest "${args[@]}"
