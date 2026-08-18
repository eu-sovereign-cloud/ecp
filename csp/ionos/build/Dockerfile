# Published ionos delegator image (ghcr.io/eu-sovereign-cloud/ecp/delegator-ionos).
#
# Single-plugin: this compiles only csp/ionos/cmd, so the image carries the ionos
# controller set and none of the aruba/dummy backends. Crossplane and the IONOS
# provider must be installed separately in the cluster for the managed resources
# this reconciles to be fulfilled (see csp/ionos/deploy).
#
# Built in workspace mode: go.work + every member's go.mod are copied so the
# workspace resolves, but only framework, resource and csp/ionos source are
# needed to compile ./cmd.

# --- Build Stage ---
FROM golang:1.26.6-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work
COPY go.work.sum go.work.sum

COPY ci/tools/go/ ci/tools/go/

# Pre-copy module manifests so replace directives resolve before source copy.
# Every go.work member must exist with a go.mod even though only framework,
# resource and csp/ionos are compiled below.
COPY framework/go.mod framework/go.sum framework/
COPY resource/go.mod resource/go.sum resource/
COPY gateway/go.mod gateway/
COPY csp/aruba/go.mod csp/aruba/go.sum csp/aruba/
COPY csp/dummy/go.mod csp/dummy/go.sum csp/dummy/
COPY csp/ionos/go.mod csp/ionos/go.sum csp/ionos/
COPY test/go.mod test/go.sum test/

WORKDIR /app/csp/ionos
RUN go mod download

WORKDIR /app
COPY framework/ framework/
COPY resource/ resource/
COPY csp/ionos/ csp/ionos/

WORKDIR /app/csp/ionos
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /delegator ./cmd

# --- Runtime Stage ---
FROM alpine:latest

COPY --from=build /delegator /delegator

# healthz/readyz
EXPOSE 8081

# Numeric and non-root: the charts set runAsNonRoot without runAsUser, so
# kubelet rejects the pod unless the image itself declares a non-zero UID.
USER 65532:65532

ENTRYPOINT ["/delegator"]
