# check=skip=InvalidDefaultArgInFrom
# Builder: codegen, lint, test, build
ARG BUILDER_BASE_IMAGE
FROM ${BUILDER_BASE_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
    && rm -rf /var/lib/apt/lists/*

# Go's -buildvcs=true (default for main packages since 1.18) shells out to git
# in the package directory. In CI, GitHub Actions mounts the checkout as the
# host runner user but runs the container as root, tripping git 2.35+
# "dubious ownership" protection (exit status 128 → "error obtaining VCS
# status" in golangci-lint / go build). Trust any path so all tools that
# invoke git (lint, gosec, govulncheck, go build) work without UID alignment.
RUN git config --system --add safe.directory '*'

WORKDIR /workspace

# Layer 1: tool definitions (change rarely — good cache hit rate)
COPY .config.mk .common.mk Makefile ./
COPY ci/tools/ ci/tools/
COPY ci/scripts/ ci/scripts/

# Install all development tools into ci/tools/bin/
RUN make tools-install
