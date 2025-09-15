#install packages for build layer
FROM golang:1.23.9-bookworm AS builder
RUN apt install git gcc make libc-dev

ARG DO_COVERAGE=false

ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.5/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.5/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

#build binary
WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

#build binary
RUN LEDGER_ENABLED=false DO_COVERAGE=${DO_COVERAGE} make install-ci

#install gex
RUN go install github.com/cosmos/gex@latest

#build main container
FROM debian:bookworm-slim
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=builder /src/injectived.sh .
COPY --from=builder /lib/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
COPY --from=builder /lib/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

RUN chmod 0644 /lib/libwasmvm.x86_64.so /lib/libwasmvm.aarch64.so

RUN apt update && apt install -y curl lz4 wget procps

RUN apt-get clean && apt-get autoclean && apt-get autoremove && rm -rf /var/lib/apt/lists/\* /tmp/\* /var/tmp/*

#configure container
VOLUME /apps/data
WORKDIR /apps/data
RUN mkdir -p /apps/data/coverage

EXPOSE 26657 26656 10337 9900 9091 9999 8545 8546

#default command
CMD ["sh", "/injectived.sh"]
