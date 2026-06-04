#!/bin/bash

SCENARIO=$1

if [[ -z $SCENARIO ]]; then
    SCENARIO=$SCENARIOS_FILTER
fi

./secatest run \
    --scenarios.filter=$SCENARIO \
    --provider.region.v1=$PROVIDER_REGION_V1 \
    --client.auth.token=$CLIENT_AUTH_TOKEN \
    --client.region=$CLIENT_REGION \
    --client.tenant=$CLIENT_TENANT