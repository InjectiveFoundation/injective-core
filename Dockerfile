#install packages for build layer
FROM golang:1.19-alpine as builder
RUN apk add --no-cache git gcc make libc-dev linux-headers
RUN set -eux; apk add --no-cache ca-certificates build-base

ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.3/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.3/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep d6904bc0082d6510f1e032fc1fd55ffadc9378d963e199afe0f93dd2667c0160
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep bb8ffda690b15765c396266721e45516cb3021146fd4de46f7daeda5b0d82c86

#Set architecture
RUN apk --print-arch > ./architecture
RUN cp /lib/libwasmvm_muslc.$(cat ./architecture).a /lib/libwasmvm_muslc.a
RUN rm ./architecture

#build binary
WORKDIR /src
COPY go.mod .
COPY go.sum .
ENV GO111MODULE=on
RUN go mod download
COPY . .

#build binary
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make install-ci

#install gex
RUN go install github.com/cosmos/gex@latest

#build main container
FROM alpine:latest
RUN apk add --no-cache ca-certificates curl tree python3 py3-pip
RUN pip3 install --upgrade pip && \
    pip3 install --no-cache-dir awscli && \
    rm -rf /var/cache/apk/*
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=builder /src/injectived.sh .

#configure container
VOLUME /apps/data
WORKDIR /apps/data
EXPOSE 26657 26656 10337 9900 9091

#default command
CMD sh /injectived.sh
