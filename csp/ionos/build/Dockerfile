# --- Build Stage ---
FROM golang:1.26.3-alpine AS build

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
COPY csp/aruba/go.mod csp/aruba/go.mod
COPY csp/aruba/go.sum csp/aruba/go.sum
COPY csp/dummy/go.mod csp/dummy/go.mod
COPY csp/dummy/go.sum csp/dummy/go.sum
COPY csp/e2e/go.mod csp/e2e/go.mod
COPY csp/e2e/go.sum csp/e2e/go.sum
COPY csp/ionos/go.mod csp/ionos/go.mod
COPY csp/ionos/go.sum csp/ionos/go.sum

WORKDIR /app/csp/ionos
RUN go mod download

# Copy full source after deps to leverage layer cache
WORKDIR /app
COPY . .

WORKDIR /app/csp/ionos
ARG BUILD_SKIP_TESTS=""
RUN [ -n "$BUILD_SKIP_TESTS" ] || go test ./...

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /plugin ./cmd

# --- Runtime Stage ---
FROM alpine:latest

COPY --from=build /plugin /plugin

ENTRYPOINT ["/plugin"]
