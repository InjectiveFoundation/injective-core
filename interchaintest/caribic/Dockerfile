# Dockerfile for Cardano IBC caribic integration tests.
FROM rust:1.88-bookworm

SHELL ["/bin/bash", "-c"]
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/cargo/bin:/usr/local/bin:${PATH}"

ARG CARDANO_IBC_REPO=https://github.com/InjectiveLabs/cardano-ibc-incubator.git
ARG CARDANO_IBC_BRANCH=v8.x-inj
ARG CARDANO_IBC_REF=110e3155016b238c86fc86f58cbed544219b52d7
ARG AIKEN_VERSION=1.1.22
ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    build-essential \
    ca-certificates \
    clang \
    cmake \
    curl \
    git \
    gnupg \
    golang-go \
    jq \
    libclang-dev \
    libprotobuf-dev \
    libssl-dev \
    make \
    netcat-openbsd \
    perl \
    pkg-config \
    protobuf-compiler \
    tar \
    unzip \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc \
    && chmod a+r /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable" > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends docker-ce-cli docker-buildx-plugin docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

ENV DENO_INSTALL=/usr/local
RUN curl -fsSL https://deno.land/install.sh | sh

RUN set -eux; \
    case "${TARGETARCH:-amd64}" in \
        amd64) AIKEN_TARGET="x86_64-unknown-linux-musl" ;; \
        arm64) AIKEN_TARGET="aarch64-unknown-linux-musl" ;; \
        *) echo "Unsupported Docker target architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL -o /tmp/aiken.tar.gz "https://github.com/aiken-lang/aiken/releases/download/v${AIKEN_VERSION}/aiken-${AIKEN_TARGET}.tar.gz"; \
    tar -xzf /tmp/aiken.tar.gz -C /tmp; \
    install -m 0755 "/tmp/aiken-${AIKEN_TARGET}/aiken" /usr/local/bin/aiken; \
    rm -rf /tmp/aiken.tar.gz "/tmp/aiken-${AIKEN_TARGET}"; \
    aiken --version

RUN git clone --branch "${CARDANO_IBC_BRANCH}" --single-branch "${CARDANO_IBC_REPO}" /opt/cardano-ibc-incubator \
    && cd /opt/cardano-ibc-incubator \
    && git checkout "${CARDANO_IBC_REF}" \
    && git submodule update --init --recursive relayer

RUN cd /opt/cardano-ibc-incubator/caribic \
    && cargo install --locked --path . --root /usr/local

RUN cd /opt/cardano-ibc-incubator/relayer \
    && cargo build --release --bin hermes \
    && ln -sf /opt/cardano-ibc-incubator/relayer/target/release/hermes /usr/local/bin/hermes

WORKDIR /opt/cardano-ibc-incubator

CMD ["tail", "-f", "/dev/null"]
