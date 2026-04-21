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

# Docker CLI (static binary) for docker-in-docker via socket mount.
# Installed as docker-bin; the docker wrapper (below) patches `docker info`
# output so KIND's rootless check passes on Podman cgroupv2.
ARG DOCKER_CLI_VERSION
RUN curl -fsSL "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_CLI_VERSION}.tgz" \
    | tar -xz --strip-components=1 -C /usr/local/bin docker/docker \
    && mv /usr/local/bin/docker /usr/local/bin/docker-bin

COPY ci/container/tools/docker-wrapper.sh /usr/local/bin/docker
RUN chmod +x /usr/local/bin/docker

# KIND (Kubernetes In Docker) for integration tests
ARG KIND_VERSION
RUN curl -fsSL -o /usr/local/bin/kind \
    "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-amd64" \
    && chmod +x /usr/local/bin/kind

# kubectl for cluster management in e2e scripts
ARG KUBECTL_VERSION
RUN curl -fsSL -o /usr/local/bin/kubectl \
    "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
    && chmod +x /usr/local/bin/kubectl

# GitHub CLI for `make branch-rebase-verify` and interactive PR workflows
ARG GH_CLI_VERSION
RUN curl -fsSL -o /tmp/gh.tgz \
    "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" \
    && tar -xzf /tmp/gh.tgz -C /tmp \
    && mv "/tmp/gh_${GH_CLI_VERSION}_linux_amd64/bin/gh" /usr/local/bin/gh \
    && rm -rf /tmp/gh.tgz "/tmp/gh_${GH_CLI_VERSION}_linux_amd64"

# Enable prompt coloring, aliases, and completion for all interactive shells
# /etc/profile.d/ is sourced by login shells (SSH sessions)
# /etc/bash.bashrc is sourced by non-login interactive shells (docker exec)

COPY ci/container/tools/shell-init.sh /etc/profile.d/ecp-tools.sh
RUN echo '. /etc/profile.d/ecp-tools.sh' >> /etc/bash.bashrc

WORKDIR /workspace
