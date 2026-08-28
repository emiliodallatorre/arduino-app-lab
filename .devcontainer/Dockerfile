FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

ARG NODE_VERSION=18.15.0
ARG YARN_VERSION=3.5.0
ARG GO_VERSION=1.26.1

ENV DEBIAN_FRONTEND=noninteractive
ENV GOPATH=/usr/local/share/go
ENV PATH=/usr/local/go/bin:${GOPATH}/bin:${PATH}

# Build and runtime dependencies for App Lab + Wails workflows.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        adb \
        build-essential \
        ca-certificates \
        curl \
        direnv \
        git \
        jq \
        libgtk-3-dev \
        libwebkit2gtk-4.1-dev \
        pkg-config \
        unzip \
        wget \
        xz-utils \
        gh && \
    rm -rf /var/lib/apt/lists/*

# Install pinned Go toolchain.
RUN arch="$(dpkg --print-architecture)" && \
    case "$arch" in \
        amd64) go_arch="amd64" ;; \
        arm64) go_arch="arm64" ;; \
        *) echo "Unsupported architecture: $arch" && exit 1 ;; \
    esac && \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz" -o /tmp/go.tgz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf /tmp/go.tgz && \
    rm -f /tmp/go.tgz

# Install pinned Node.js and Yarn (Berry) used by this repository.
RUN arch="$(dpkg --print-architecture)" && \
    case "$arch" in \
        amd64) node_arch="x64" ;; \
        arm64) node_arch="arm64" ;; \
        *) echo "Unsupported architecture: $arch" && exit 1 ;; \
    esac && \
    curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.xz" -o /tmp/node.tar.xz && \
    rm -rf /usr/local/lib/node_modules && \
    tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1 && \
    rm -f /tmp/node.tar.xz && \
    corepack enable && \
    corepack prepare "yarn@${YARN_VERSION}" --activate

# Install Wails CLI for desktop app development.
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest && \
    ln -sf ${GOPATH}/bin/wails /usr/local/bin/wails

RUN mkdir -p ${GOPATH} && chown -R vscode:vscode ${GOPATH}

USER vscode
