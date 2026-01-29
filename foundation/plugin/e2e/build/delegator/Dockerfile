# Build the manager binary
FROM golang:1.24-bookworm as builder

WORKDIR /workspace

# Install dlv for debugging
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Copy all the sources
COPY . .

# Build the delegator with debug information
WORKDIR /workspace/foundation/plugin/e2e
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="all=-N -l" -o delegator cmd/delegator/main.go

# Use a more hacking-friendly base image for easier debugging and access
FROM debian:bookworm-slim

# Install necessary debugging tools and a shell
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    curl \
    ca-certificates \
    less \
    gnupg \
    procps \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*

# Copy the manager binary from the builder stage
WORKDIR /app
COPY --from=builder /workspace/foundation/plugin/e2e/delegator .

# Copy dlv (Delve debugger) for debugging. Assume dlv is in /go/bin in the builder image.
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv

# Expose default Delve debugger port
EXPOSE 40000
# Expose healthz port
EXPOSE 8081

# Create a non-root user with a specific UID and switch to it
RUN groupadd -r -g 1001 appuser && useradd -r -u 1001 -g appuser appuser
USER 1001

ENTRYPOINT ["/app/delegator"]