# check=skip=InvalidDefaultArgInFrom
# Tools: interactive shell with completion, coloring, vim
ARG BUILDER_IMAGE
FROM ${BUILDER_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash-completion \
    curl \
    vim \
    less \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Docker CLI (static binary) for docker-in-docker via socket mount
ARG DOCKER_CLI_VERSION
RUN curl -fsSL "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_CLI_VERSION}.tgz" \
    | tar -xz --strip-components=1 -C /usr/local/bin docker/docker

# Enable prompt coloring, aliases, and completion for all interactive shells
# /etc/profile.d/ is sourced by login shells (SSH sessions)
# /etc/bash.bashrc is sourced by non-login interactive shells (docker exec)

COPY ci/container/tools/shell-init.sh /etc/profile.d/ecp-tools.sh
RUN echo '. /etc/profile.d/ecp-tools.sh' >> /etc/bash.bashrc

WORKDIR /workspace
