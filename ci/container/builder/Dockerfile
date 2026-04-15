# check=skip=InvalidDefaultArgInFrom
# Builder: codegen, lint, test, build
ARG BUILDER_BASE_IMAGE
FROM ${BUILDER_BASE_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
    python3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Layer 1: tool definitions (change rarely — good cache hit rate)
COPY .config.mk .common.mk Makefile ./
COPY ci/tools/ ci/tools/
COPY ci/scripts/ ci/scripts/

# Install all development tools into ci/tools/bin/
RUN make tools-install
