# --- Build Stage ---
FROM golang:1.26.2-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

# Copy go.work files first for workspace resolution
COPY go.work go.work
COPY go.work.sum go.work.sum

COPY ci/tools/go/ ci/tools/go/

# Pre-copy module manifests so replace directives resolve before full source copy
COPY foundation/persistence/go.mod foundation/persistence/go.mod
COPY foundation/persistence/go.sum foundation/persistence/go.sum
COPY foundation/delegator/go.mod foundation/delegator/go.mod
COPY foundation/delegator/go.sum foundation/delegator/go.sum
COPY foundation/gateway/go.mod foundation/gateway/go.mod
COPY foundation/gateway/go.sum foundation/gateway/go.sum
COPY foundation/plugin/aruba/go.mod foundation/plugin/aruba/go.mod
COPY foundation/plugin/aruba/go.sum foundation/plugin/aruba/go.sum
COPY foundation/plugin/dummy/go.mod foundation/plugin/dummy/go.mod
COPY foundation/plugin/dummy/go.sum foundation/plugin/dummy/go.sum
COPY foundation/plugin/e2e/go.mod foundation/plugin/e2e/go.mod
COPY foundation/plugin/e2e/go.sum foundation/plugin/e2e/go.sum
COPY foundation/plugin/ionos/go.mod foundation/plugin/ionos/go.mod
COPY foundation/plugin/ionos/go.sum foundation/plugin/ionos/go.sum

WORKDIR /app/foundation/plugin/ionos
RUN go mod download

# Copy full source after deps to leverage layer cache
WORKDIR /app
COPY . .

WORKDIR /app/foundation/plugin/ionos
ARG BUILD_SKIP_TESTS=""
RUN [ -n "$BUILD_SKIP_TESTS" ] || go test ./...

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /plugin ./cmd

# --- Runtime Stage ---
FROM alpine:latest

COPY --from=build /plugin /plugin

ENTRYPOINT ["/plugin"]
