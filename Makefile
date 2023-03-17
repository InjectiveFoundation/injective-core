APP_VERSION = $(shell git describe --abbrev=0 --tags)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u "+%Y%m%d-%H%M")
COSMOS_VERSION_PKG = github.com/cosmos/cosmos-sdk/version
COSMOS_VERSION_NAME = injective
VERSION_PKG = github.com/InjectiveLabs/injective-core/version
PACKAGES=$(shell go list ./... | grep -Ev 'vendor|importer|gen|api/design|rpc/tester')
IMAGE_NAME := gcr.io/injective-core/core

# process build tags
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq ($(WITH_CLEVELDB),yes)
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))
whitespace :=
empty = $(whitespace) $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(empty),$(comma),$(build_tags))

all:

image:
	docker build --build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(IMAGE_NAME):local -f Dockerfile .
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):$(GIT_COMMIT)
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):latest

push:
	docker push $(IMAGE_NAME):$(GIT_COMMIT)
	docker push $(IMAGE_NAME):latest

install: export GOPROXY=direct
install: export VERSION_FLAGS="-X $(VERSION_PKG).AppVersion=$(APP_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)  -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=injectived -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install:
	cd cmd/injectived/ && go install -tags $(build_tags_comma_sep) $(BUILD_FLAGS) -ldflags $(VERSION_FLAGS)

install-ci: export GOPROXY=https://goproxy.injective.dev,direct
install-ci: export VERSION_FLAGS="-X $(VERSION_PKG).AppVersion=$(APP_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)  -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=injectived -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install-ci:
	cd cmd/injectived/ && go install -tags $(build_tags_comma_sep) $(BUILD_FLAGS) -ldflags $(VERSION_FLAGS)

.PHONY: install image push gen lint test mock cover

mock: export GOPROXY=direct
mock: tests/mocks.go
	go install github.com/golang/mock/mockgen
	go generate ./tests/...

PKGS_TO_COVER := $(shell go list ./injective-chain/modules/exchange | paste -sd "," -)

deploy:
	./deploy_contracts.sh

fuzz:
	go test -fuzz FuzzTest ./injective-chain/modules/exchange/testexchange/fuzztesting

test: export GOPROXY=direct
test:
	go install github.com/onsi/ginkgo/ginkgo@latest
	ginkgo -r --race --randomizeSuites --randomizeAllSpecs --coverpkg=$(PKGS_TO_COVER) ./...

test-erc20bridge:
	@go test -v ./injective-chain/modules/erc20bridge/...
test-exchange:
	@go test -v ./injective-chain/modules/exchange/...
test-unit:
	@go test -v ./... $(PACKAGES)

test-rpc:
	MODE="rpc" go test -v ./tests/...

lint: export GOPROXY=direct
lint:
	golangci-lint run

cover:
	go tool cover -html=tests/injective-chain/modules/exchange/exchange.coverprofile

build-release-%: export TARGET=$*
build-release-%: export DOCKER_BUILDKIT=1
build-release-%: export VERSION_FLAGS="-X $(VERSION_PKG).AppVersion=$(APP_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)"
build-release-%:
	docker build \
		--build-arg LDFLAGS=$(VERSION_FLAGS) \
		--build-arg PKG=github.com/InjectiveLabs/injective-core/cmd/$(TARGET) \
		--ssh=default -t $(TARGET)-release -f Dockerfile.release .

prepare-release-%: export TARGET=$*
prepare-release-%:
	mkdir -p dist/$(TARGET)_linux_amd64/
	mkdir -p dist/$(TARGET)_darwin_amd64/
	mkdir -p dist/$(TARGET)_windows_amd64/
	#
	docker create --name tmp_$(TARGET) $(TARGET)-release bash
	#
	docker cp tmp_$(TARGET):/root/go/bin/$(TARGET)-linux-amd64 dist/$(TARGET)_linux_amd64/$(TARGET)
	docker cp tmp_$(TARGET):/root/go/bin/$(TARGET)-darwin-amd64 dist/$(TARGET)_darwin_amd64/$(TARGET)
	docker cp tmp_$(TARGET):/root/go/bin/$(TARGET)-windows-amd64 dist/$(TARGET)_windows_amd64/$(TARGET).exe
	#
	docker rm tmp_$(TARGET)

mongo:
	mkdir -p ./var/mongo
	mongod --dbpath ./var/mongo

###############################################################################
###                                Protobuf                                 ###
###############################################################################

gen:
	@echo "Generating Protobuf files without ts bindings."
	@echo "👉 Run make gen-all if you want to generate the files with ts bindings"
	@echo "\n"
	@./scripts/protocgen.sh
	@./scripts/protoc-swagger-gen.sh

gen-all:
	@./scripts/protocgen.sh
	@./scripts/protoc-swagger-gen.sh

gen-ts:
	@./client/proto-ts/scripts/gen-proto-ts.sh
	@./client/proto-ts/scripts/gen-proto-ts-types-ignore.sh
	@./client/proto-ts/scripts/gen-proto-grpc-web-package.sh

publish-ts:
	@./client/proto-ts/scripts/gen-proto-ts-publish.sh
	@./client/proto-ts/scripts/gen-proto-ts-publish-esm.sh

grpc-ui:
	grpcui -plaintext -protoset ./injectived.protoset localhost:9900

proto-all: proto-gen proto-swagger-gen proto-format

proto-gen:
	@./scripts/protocgen.sh

proto-format:
	find ./ -not -path "./third_party/*" -name *.proto -exec clang-format -i {} \;

proto-swagger-gen:
	@./scripts/protoc-swagger-gen.sh

proto-lint:
	@buf check lint --error-format=json

proto-check-breaking:
	@buf check breaking --against-input '.git#branch=development'

proto-lint-docker:
	@$(DOCKER_BUF) check lint --error-format=json
.PHONY: proto-lint

proto-check-breaking-docker:
	@$(DOCKER_BUF) check breaking --against-input $(HTTPS_GIT)#branch=development
.PHONY: proto-check-breaking-ci

TM_URL           = https://raw.githubusercontent.com/tendermint/tendermint/v0.34.0-rc4/proto/tendermint
GOGO_PROTO_URL   = https://raw.githubusercontent.com/regen-network/protobuf/cosmos
COSMOS_PROTO_URL = https://raw.githubusercontent.com/regen-network/cosmos-proto/master
COSMOS_SDK_URL = https://raw.githubusercontent.com/cosmos/cosmos-sdk/master
CONFIO_URL 		 = https://raw.githubusercontent.com/confio/ics23/v0.6.2

TM_CRYPTO_TYPES     = third_party/proto/tendermint/crypto
TM_ABCI_TYPES       = third_party/proto/tendermint/abci
TM_TYPES     			  = third_party/proto/tendermint/types
TM_VERSION 					= third_party/proto/tendermint/version
TM_LIBS							= third_party/proto/tendermint/libs/bits

GOGO_PROTO_TYPES    = third_party/proto/gogoproto
COSMOS_PROTO_TYPES  = third_party/proto/cosmos_proto
CONFIO_TYPES        = third_party/proto/confio

COSMOS_SDK_PROTO  = third_party/proto/cosmos-sdk

proto-update-deps:
	@mkdir -p $(GOGO_PROTO_TYPES)
	@curl -sSL $(GOGO_PROTO_URL)/gogoproto/gogo.proto > $(GOGO_PROTO_TYPES)/gogo.proto

	@mkdir -p $(COSMOS_PROTO_TYPES)
	@curl -sSL $(COSMOS_PROTO_URL)/cosmos.proto > $(COSMOS_PROTO_TYPES)/cosmos.proto

## Importing of tendermint protobuf definitions currently requires the
## use of `sed` in order to build properly with cosmos-sdk's proto file layout
## (which is the standard Buf.build FILE_LAYOUT)
## Issue link: https://github.com/tendermint/tendermint/issues/5021
	@mkdir -p $(TM_ABCI_TYPES)
	@curl -sSL $(TM_URL)/abci/types.proto > $(TM_ABCI_TYPES)/types.proto

	@mkdir -p $(TM_VERSION)
	@curl -sSL $(TM_URL)/version/types.proto > $(TM_VERSION)/types.proto

	@mkdir -p $(TM_TYPES)
	@curl -sSL $(TM_URL)/types/types.proto > $(TM_TYPES)/types.proto
	@curl -sSL $(TM_URL)/types/evidence.proto > $(TM_TYPES)/evidence.proto
	@curl -sSL $(TM_URL)/types/params.proto > $(TM_TYPES)/params.proto
	@curl -sSL $(TM_URL)/types/validator.proto > $(TM_TYPES)/validator.proto

	@mkdir -p $(TM_CRYPTO_TYPES)
	@curl -sSL $(TM_URL)/crypto/proof.proto > $(TM_CRYPTO_TYPES)/proof.proto
	@curl -sSL $(TM_URL)/crypto/keys.proto > $(TM_CRYPTO_TYPES)/keys.proto

	@mkdir -p $(TM_LIBS)
	@curl -sSL $(TM_URL)/libs/bits/types.proto > $(TM_LIBS)/types.proto

	@mkdir -p $(CONFIO_TYPES)
	@curl -sSL $(CONFIO_URL)/proofs.proto > $(CONFIO_TYPES)/proofs.proto


.PHONY: proto-all proto-gen proto-lint proto-check-breaking proto-update-deps


###############################################################################
###                              Documentation                              ###
###############################################################################

update-swagger-docs:
	statik -src=client/docs/swagger-ui -dest=client/docs -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
    	echo "\033[92mSwagger docs are in sync\033[0m";\
    fi
.PHONY: update-swagger-docs
