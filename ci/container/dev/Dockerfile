# check=skip=InvalidDefaultArgInFrom
# Dev: SSH server, editor language servers
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE}

RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-server \
    neovim \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Go language server (gopls) for neovim, VSCode, GoLand
RUN go install golang.org/x/tools/gopls@latest \
    && mv "$(go env GOPATH)/bin/gopls" /usr/local/bin/gopls \
    && rm -rf "$(go env GOPATH)"

# sshd runtime directory
RUN mkdir -p /run/sshd

COPY ci/container/dev/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
