#!/bin/bash

set -ue

DESTDIR=${DESTDIR:-}
PREFIX=${PREFIX:-/usr/local}
UNAME_S="$(uname -s 2>/dev/null)"
UNAME_M="$(uname -m 2>/dev/null)"
BUF_VERSION=0.11.0
PROTOC_VERSION=3.15.8
PROTOC_GRPC_GATEWAY_VERSION=1.16.0

f_abort() {
    local l_rc=$1
    shift
    
    echo $@ >&2
    exit ${l_rc}
}

case "${UNAME_S}" in
    Linux)
        PROTOC_ZIP="protoc-${PROTOC_VERSION}-linux-x86_64.zip"
    ;;
    Darwin)
        PROTOC_ZIP="protoc-${PROTOC_VERSION}-osx-x86_64.zip"
    ;;
    *)
        f_abort 1 "Unknown kernel name. Exiting."
esac

TEMPDIR="$(mktemp -d)"

trap "rm -rvf ${TEMPDIR}" EXIT

f_print_installing_with_padding() {
    printf "Installing %30s ..." "$1" >&2
}

f_print_done() {
    echo -e "\tDONE" >&2
}

f_ensure_tools() {
    ! which curl &>/dev/null && f_abort 2 "couldn't find curl, aborting" || true
}

f_ensure_dirs() {
    mkdir -p "${DESTDIR}/${PREFIX}/bin"
    mkdir -p "${DESTDIR}/${PREFIX}/include"
}

f_needs_install() {
    if which $1 &>/dev/null ; then
        echo -e "\talready installed. Skipping."
        return 1
    fi

    return 0
}

f_install_protoc() {
    f_print_installing_with_padding proto_c
    f_needs_install "protoc" || return 0

    pushd "${TEMPDIR}" >/dev/null
    curl -o "${PROTOC_ZIP}" -sSL "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}"
    unzip -q -o ${PROTOC_ZIP} -d ${DESTDIR}/${PREFIX} bin/protoc; \
    unzip -q -o ${PROTOC_ZIP} -d ${DESTDIR}/${PREFIX} 'include/*'; \
    rm -f ${PROTOC_ZIP}
    popd >/dev/null
    f_print_done
}

f_install_buf() {
    f_print_installing_with_padding buf
    f_needs_install "buf" || return 0

    curl -sSL "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-${UNAME_S}-${UNAME_M}" -o "${DESTDIR}/${PREFIX}/bin/buf"
    chmod +x "${DESTDIR}/${PREFIX}/bin/buf"
    f_print_done
}

f_install_protoc_gen_gocosmos() {
    f_print_installing_with_padding protoc-gen-gocosmos
    f_needs_install "protoc-gen-gocosmos" || return 0

    if ! grep "github.com/cosmos/gogoproto => github.com/regen-network/protobuf" go.mod &>/dev/null ; then
        echo -e "\tPlease run this command from somewhere inside the ethermint folder."
        return 1
    fi

    pushd "${TEMPDIR}" >/dev/null
    curl -L -O https://github.com/regen-network/cosmos-proto/archive/refs/heads/master.zip && unzip master.zip
    cd cosmos-proto-master/protoc-gen-gocosmos && go install .
    cd ../.. && rm -fr master.zip gogoproto-master
    popd >/dev/null
    f_print_done
}

f_install_protoc_gen_grpc_gateway() {
    f_print_installing_with_padding protoc-gen-grpc-gateway
    f_needs_install "protoc-gen-grpc-gateway" || return 0

    go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v${PROTOC_GRPC_GATEWAY_VERSION}
    f_print_done
}

f_install_protoc_gen_swagger() {
    f_print_installing_with_padding protoc-gen-swagger
    f_needs_install "protoc-gen-swagger" || return 0
    echo

    if ! which npm &>/dev/null ; then
        echo -e "\tNPM is not installed. Skipping."
        return 0
    fi

    pushd "${TEMPDIR}" >/dev/null
    go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v${PROTOC_GRPC_GATEWAY_VERSION}
    go install github.com/rakyll/statik@latest
    npm install -g swagger-combine
    popd >/dev/null
    f_print_done
}

f_install_protoc_gen_ts() {
    f_print_installing_with_padding protoc-gen-ts
    f_needs_install "protoc-gen-ts" || return 0
    echo

    if ! which npm &>/dev/null ; then
        echo -e "\tNPM is not installed. Skipping."
        return 0
    fi

    pushd "${TEMPDIR}" >/dev/null
    npm install -g ts-protoc-gen@next
    popd >/dev/null
    f_print_done
}

f_install_clang_format() {
    f_print_installing_with_padding clang-format

    f_needs_install "clang-format" || return 0

    case "${UNAME_S}" in
        Linux)
            if [ -e /etc/debian_version ]; then
                echo -e "\tRun: sudo apt-get install clang-format" >&2
                elif [ -e /etc/fedora-release ]; then
                echo -e "\tRun: sudo dnf install clang" >&2
            else
                echo -e "\tRun (as root): subscription-manager repos --enable rhel-7-server-devtools-rpms ; yum install llvm-toolset-7" >&2
            fi
        ;;
        Darwin)
            echo "\tRun: brew install clang-format" >&2
        ;;
        *)
            echo "\tunknown operating system. Skipping." >&2
    esac
}

f_ensure_tools
f_ensure_dirs
f_install_protoc
f_install_buf
f_install_protoc_gen_gocosmos
f_install_protoc_gen_grpc_gateway
f_install_protoc_gen_swagger
f_install_protoc_gen_ts
#f_install_clang_format
