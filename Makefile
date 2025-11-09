APP_VERSION = $(shell git describe --tags --match "v*")
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u "+%Y%m%d-%H%M")
COSMOS_VERSION_PKG = github.com/cosmos/cosmos-sdk/version
COSMOS_VERSION_NAME = injective
INJECTIVED_VERSION_PKG = github.com/InjectiveLabs/injective-core/version
PEGGO_VERSION_PKG = github.com/InjectiveLabs/injective-core/peggo/orchestrator/version
IMAGE_NAME := injectivelabs/injective-core
GOPROXY ?= https://goproxy.injective.dev,direct
LEDGER_ENABLED ?= true

ifeq ($(DO_COVERAGE),true)
coverage_flags_injectived = -coverpkg=`cat pkgs-injectived.txt`
coverage_flags_peggo = -coverpkg=`cat pkgs-peggo.txt`
else ifeq ($(DO_COVERAGE),yes)
coverage_flags_injectived = -coverpkg=`cat pkgs-injectived.txt`
coverage_flags_peggo = -coverpkg=`cat pkgs-peggo.txt`
else
coverage_flags_injectived =
coverage_flags_peggo =
endif

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

init:
	@git config core.hooksPath .github/hooks

image:
	docker build --build-arg DO_COVERAGE=$(DO_COVERAGE) --build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(IMAGE_NAME):local -f Dockerfile .
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):$(GIT_COMMIT)
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):latest

push:
	docker push $(IMAGE_NAME):$(GIT_COMMIT)
	docker push $(IMAGE_NAME):latest

pkgs-injectived.txt:
	@go list \
		-f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		-deps ./cmd/injectived | \
	grep -E '^(cosmossdk\.io/|github\.com/bandprotocol/|github\.com/cometbft/|github\.com/cosmos/|github\.com/CosmWasm/|github\.com/ethereum/|github\.com/InjectiveLabs/)' | \
	paste -d "," -s - > pkgs-injectived.txt

pkgs-peggo.txt:
	@go list \
		-f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		-deps ./cmd/peggo | \
	grep -E '^(cosmossdk\.io/|github\.com/bandprotocol/|github\.com/cometbft/|github\.com/cosmos/|github\.com/CosmWasm/|github\.com/ethereum/|github\.com/InjectiveLabs/)' | \
	paste -d "," -s - > pkgs-peggo.txt

install-injectived: export VERSION_FLAGS="-X $(INJECTIVED_VERSION_PKG).AppVersion=$(APP_VERSION) -X $(INJECTIVED_VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(INJECTIVED_VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=injectived -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install-injectived:
	@echo "Installing injectived..."
	go install -tags $(build_tags_comma_sep) -ldflags $(VERSION_FLAGS) $(coverage_flags_injectived) ./cmd/injectived

install-peggo: export VERSION_FLAGS="-X $(PEGGO_VERSION_PKG).AppVersion=$(APP_VERSION) -X $(PEGGO_VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(PEGGO_VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=peggo -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install-peggo:
	@echo "Installing peggo..."
	go install -ldflags $(VERSION_FLAGS) $(coverage_flags_peggo) ./cmd/peggo

# install is used for local development
# doesn't support DO_COVERAGE, but enforces git hooks
# for DO_COVERAGE, use install-ci that pre-fills pkgs-*.txt
install: init
install: install-injectived install-peggo
install:
	@echo "Installed injectived and peggo"

# install-ci is used for the CI pipeline
# pre-fills pkgs-*.txt for DO_COVERAGE
install-ci: pkgs-injectived.txt pkgs-peggo.txt
install-ci: install-injectived install-peggo
install-ci:
	@echo "Installed injectived and peggo"
	@rm pkgs-injectived.txt pkgs-peggo.txt

.PHONY: init install install-ci install-injectived install-peggo
.PHONY: image push gen lint lint-last-commit test mock cover

mock: tests/mocks.go
	go install github.com/golang/mock/mockgen@latest
	go generate ./tests/...

PKGS_TO_COVER := $(shell go list ./injective-chain/modules/exchange | paste -sd "," -)

###############################################################################
###                                   testing                               ###
###############################################################################
test: ictest-all test-unit

test-unit:
	go install github.com/onsi/ginkgo/ginkgo@latest
	ginkgo -r --race --randomizeSuites --randomizeAllSpecs --coverpkg=$(PKGS_TO_COVER) ./...

test-fuzz: # use old clang linker on macOS https://github.com/golang/go/issues/65169
	go test -fuzz FuzzTest ./injective-chain/modules/exchange/testexchange/fuzztesting -ldflags=-extldflags=-Wl,-ld_classic

test-exchange:
	go test -race -v ./injective-chain/modules/exchange/...

test-rpc:
	MODE="rpc" go test -v ./tests/...

cover:
	go tool cover -html=tests/injective-chain/modules/exchange/exchange.coverprofile

.PHONY: test test-unit test-fuzz test-exchange test-rpc

# TODO: add runsim and benchmarking

###############################################################################
###                             e2e interchain test                         ###
###############################################################################

rm-testcache:
	go clean -testcache

rm-ic-coverage:
	rm -rf interchaintest/coverage

ictest-all: rm-testcache rm-ic-coverage
	cd interchaintest && go test -timeout 60m -v -run ./...

ictest-basic: rm-testcache
	rm -rf interchaintest/coverage/TestBasicInjectiveStart
	cd interchaintest && go test -v -run TestBasicInjectiveStart .
	./scripts/coverage-html.sh interchaintest/coverage/TestBasicInjectiveStart

ictest-upgrade: rm-testcache
	rm -rf interchaintest/coverage/TestInjectiveUpgradeHandler
	cd interchaintest && go test -v -run TestInjectiveUpgradeHandler .
	./scripts/coverage-html.sh interchaintest/coverage/TestInjectiveUpgradeHandler

ictest-dynamic-fee: rm-testcache
	rm -rf interchaintest/coverage/TestDynamicFee
	cd interchaintest && go test -v -run Test_DynamicFee .
	./scripts/coverage-html.sh interchaintest/coverage/TestDynamicFee

ictest-ibchooks: rm-testcache
	rm -rf interchaintest/coverage/TestInjectiveIBCHooks
	cd interchaintest && go test -v -run TestInjectiveIBCHooks .
	./scripts/coverage-html.sh interchaintest/coverage/TestInjectiveIBCHooks

ictest-permissions-wasm-hook: rm-testcache
	rm -rf interchaintest/coverage/TestPermissionedDenomWasmHookCall
	cd interchaintest && go test -v -run TestPermissionedDenomWasmHookCall .
	./scripts/coverage-html.sh interchaintest/coverage/TestPermissionedDenomWasmHookCall

ictest-pfm: rm-testcache
	rm -rf interchaintest/coverage/TestPacketForwardMiddleware
	cd interchaintest && go test -v -run TestPacketForwardMiddleware .
	./scripts/coverage-html.sh interchaintest/coverage/TestPacketForwardMiddleware

ictest-lanes: rm-testcache
	rm -rf interchaintest/coverage/TestLanes
	cd interchaintest && go test -v -run MempoolLanes .
	./scripts/coverage-html.sh interchaintest/coverage/TestLanes

ictest-fixed-gas: rm-testcache
	rm -rf interchaintest/coverage/Test_FixedGas_HappyPath
	cd interchaintest && go test -timeout 30m -v -run Test_FixedGas_HappyPath .
	./scripts/coverage-html.sh interchaintest/coverage/Test_FixedGas_HappyPath

ictest-fixed-gas-regression: rm-testcache
	rm -rf interchaintest/coverage/Test_FixedGas_Regression
	cd interchaintest && go test -timeout 30m -v -run Test_FixedGas_Regression .
	./scripts/coverage-html.sh interchaintest/coverage/Test_FixedGas_Regression

ictest-peggo: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_Basic
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_Basic .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_Basic

ictest-peggo-ibc: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_IBCDenomDeployed
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_IBCDenomDeployed .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_IBCDenomDeployed

ictest-peggo-erc20: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_ERC20DenomDeployed
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_ERC20DenomDeployed .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_ERC20DenomDeployed

ictest-evm: rm-testcache
	rm -rf interchaintest/coverage/TestEVMRPC
	cd interchaintest && go test -v -run "(EVMRPC*|EVMKeeper*)" .
	./scripts/coverage-html.sh interchaintest/coverage/TestEVMRPC

ictest-chainstream: rm-testcache
	rm -rf interchaintest/coverage/Test_ChainStream_ConnectsAndReceivesEvents
	cd interchaintest && go test -timeout 30m -v -run Test_ChainStream_ConnectsAndReceivesEvents .
	./scripts/coverage-html.sh interchaintest/coverage/Test_ChainStream_ConnectsAndReceivesEvents

ictest-downtime-detector: rm-testcache
	rm -rf interchaintest/coverage/TestDowntimeDetector
	cd interchaintest && go test -timeout 30m -v -run TestDowntimeDetector .
	./scripts/coverage-html.sh interchaintest/coverage/TestDowntimeDetector

ictest-hyperlane: rm-testcache
	rm -rf interchaintest/coverage/Test_HyperLaneRemoteTransfer_CosmosNativeXCosmosNative
	cd interchaintest && go test -timeout 30m -v -run Test_HyperLaneRemoteTransfer_CosmosNativeXCosmosNative .
	./scripts/coverage-html.sh interchaintest/coverage/Test_HyperLaneRemoteTransfer_CosmosNativeXCosmosNative

ictest-validator-jailed: rm-testcache
	rm -rf interchaintest/coverage/Test_ValidatorJailedEvent
	cd interchaintest && go test -timeout 30m -v -run Test_ValidatorJailedEvent .
	./scripts/coverage-html.sh interchaintest/coverage/Test_ValidatorJailedEvent

.PHONY: rm-testcache rm-ic-coverage
.PHONY: ictest-all ictest-basic ictest-upgrade ictest-ibchooks ictest-permissions-wasm-hook ictest-pfm ictest-lanes
.PHONY: ictest-fixed-gas ictest-fixed-gas-regression ictest-peggo ictest-peggo-ibc ictest-hyperlane ictest-evm
.PHONY: ictest-downtime-detector ictest-chainstream ictest-validator-jailed

###############################################################################

lint:
	golangci-lint run --timeout=15m --new-from-rev=master

lint-last-commit:
	golangci-lint run --timeout=15m --new-from-rev=HEAD~

###############################################################################
###                                Protobuf                                 ###
###############################################################################

DOCKER=docker
protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto: proto-format proto-gen proto-swagger-gen

proto-gen:
	@$(protoImage) sh ./scripts/protocgen.sh

proto-gen-pulsar:
	@$(protoImage) sh ./scripts/protocgen-pulsar.sh

proto-swagger-gen:
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json ./proto

proto-check-breaking:
	@$(protoImage) buf breaking --against '.git#branch=main'

grpc-ui:
	grpcui -plaintext -protoset ./injectived.protoset localhost:9900

.PHONY: proto proto-gen proto-lint proto-check-breaking proto-update-deps

###############################################################################
###                           Precompiles Bindings                          ###
###############################################################################

precompiles-bindings:
	./scripts/precompiles-bindings.sh

###############################################################################
###                              Documentation                              ###
###############################################################################

gen-modules-errors-pages:
	@exec ./scripts/docs/generate_errors_docs.sh

# Default destination folder for error documentation JSON files
ERROR_DOCS_DEST ?= ./docs/errors

# Generate error documentation JSON files for all registered error codes
# Usage:
#   make gen-error-docs                                    # Generate in default location (./docs/errors)
#   make gen-error-docs ERROR_DOCS_DEST=./custom/path     # Generate in custom directory
gen-error-docs:
	@echo "Generating error documentation JSON files..."
	@mkdir -p $(ERROR_DOCS_DEST)
	@go run scripts/docs/document_error_codes_script.go -dest $(ERROR_DOCS_DEST)
	@echo "Error documentation generated successfully in $(ERROR_DOCS_DEST)"

.PHONY: gen-modules-errors-pages gen-error-docs
