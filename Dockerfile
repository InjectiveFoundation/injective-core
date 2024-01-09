#install packages for build layer
FROM golang:1.19-bookworm as builder
RUN apt install git gcc make libc-dev

ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.5.0/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.5.0/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

#build binary
WORKDIR /src
COPY go.mod .
COPY go.sum .
ENV GO111MODULE=on
RUN go mod download
COPY . .

#build binary
RUN LEDGER_ENABLED=false make install-ci

#install gex
RUN go install github.com/cosmos/gex@latest

#build main container
FROM debian:bookworm-slim
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=builder /src/injectived.sh .

RUN apt update && apt install -y curl lz4 wget procps

RUN apt-get clean && apt-get autoclean && apt-get autoremove && rm -rf /var/lib/apt/lists/\* /tmp/\* /var/tmp/*

ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.5.0/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.5.0/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

#configure container
VOLUME /apps/data
WORKDIR /apps/data
EXPOSE 26657 26656 10337 9900 9091 9999

#default command
CMD sh /injectived.sh