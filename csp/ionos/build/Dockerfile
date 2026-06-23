# --- Build Stage ---
FROM golang:1.26.4-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work
COPY go.work.sum go.work.sum

COPY ci/tools/go/ ci/tools/go/

# Pre-copy module manifests so replace directives resolve before full source copy
COPY framework/go.mod framework/go.mod
COPY framework/go.sum framework/go.sum
COPY resource/go.mod resource/go.mod
COPY csp/ionos/go.mod csp/ionos/go.mod
COPY csp/ionos/go.sum csp/ionos/go.sum
COPY modules/go-sdk/ modules/go-sdk/

WORKDIR /app/csp/ionos
RUN go mod download

# Copy full source after deps to leverage layer cache
WORKDIR /app
COPY framework/ framework/
COPY resource/ resource/
COPY csp/ionos/ csp/ionos/
COPY modules/go-sdk/ modules/go-sdk/

WORKDIR /app/csp/ionos
ARG BUILD_SKIP_TESTS=""
RUN [ -n "$BUILD_SKIP_TESTS" ] || go test ./...

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /plugin ./cmd

# --- Runtime Stage ---
FROM alpine:latest

COPY --from=build /plugin /plugin

ENTRYPOINT ["/plugin"]
