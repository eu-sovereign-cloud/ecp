# Builder: codegen, lint, test, build
ARG BUILDER_BASE_IMAGE
FROM ${BUILDER_BASE_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
