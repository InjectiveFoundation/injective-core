APP_VERSION = $(shell git describe --tags --match "v*")
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u "+%Y%m%d-%H%M")
COSMOS_VERSION_PKG = github.com/cosmos/cosmos-sdk/version
COSMOS_VERSION_NAME = injective
INJECTIVED_VERSION_PKG = github.com/InjectiveLabs/injective-core/version
PEGGO_VERSION_PKG = github.com/InjectiveLabs/injective-core/peggo/orchestrator/version
IMAGE_NAME := injectivelabs/injective-core
GOPROXY ?= https://proxy.golang.org,direct
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

image-foundry-deployer:
	docker build -t injectivelabs/injective-foundry-deployer:local -f interchaintest/foundry/Dockerfile .

image-caribic:
	docker build -t injectivelabs/injective-caribic:local -f interchaintest/caribic/Dockerfile .

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
.PHONY: image image-foundry-deployer image-caribic push gen lint lint-last-commit test mock cover

mock: tests/mocks.go
	go install github.com/golang/mock/mockgen@latest
	go generate ./tests/...

PKGS_TO_COVER := $(shell go list ./injective-chain/modules/exchange | paste -sd "," -)

###############################################################################
###                                   testing                               ###
###############################################################################
test: ictest-all test-unit

test-unit:
	go test -race -shuffle=on -coverpkg=$(PKGS_TO_COVER) ./...

FUZZTIME ?= 0
test-fuzz:
	$(if $(filter 0,$(FUZZTIME)), \
		go test -v -fuzz FuzzTest ./injective-chain/modules/exchange/testexchange/fuzztesting, \
		go test -v -fuzz FuzzTest -fuzztime $(FUZZTIME) ./injective-chain/modules/exchange/testexchange/fuzztesting)

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

ictest-cardano-ibc-client: rm-testcache
	rm -rf interchaintest/coverage/Test_CardanoProbabilisticClientCanBeCreated
	cd interchaintest && go test -timeout 30m -v -run Test_CardanoProbabilisticClientCanBeCreated .
	./scripts/coverage-html.sh interchaintest/coverage/Test_CardanoProbabilisticClientCanBeCreated

ictest-cardano-full-token-swap: rm-testcache
	rm -rf interchaintest/coverage/Test_CardanoIBC_FullTokenSwap
	cd interchaintest && go test -timeout 90m -v -run Test_CardanoIBC_FullTokenSwap .
	./scripts/coverage-html.sh interchaintest/coverage/Test_CardanoIBC_FullTokenSwap

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

ictest-fixed-gas-cross-margin: rm-testcache
	rm -rf interchaintest/coverage/Test_FixedGas_CrossMargin
	cd interchaintest && go test -timeout 30m -v -run Test_FixedGas_CrossMargin .
	./scripts/coverage-html.sh interchaintest/coverage/Test_FixedGas_CrossMargin

ictest-peggo: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_Basic
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_Basic .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_Basic

ictest-peggo-rate-limit: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_RateLimits
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_RateLimits .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_RateLimits

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

ictest-oracle-morpho: rm-testcache
	rm -rf interchaintest/coverage/Test_OraclePrecompile_MorphoWrapper
	cd interchaintest && go test -timeout 30m -v -run Test_OraclePrecompile_MorphoWrapper .
	./scripts/coverage-html.sh interchaintest/coverage/Test_OraclePrecompile_MorphoWrapper

ictest-chainstream: rm-testcache
	rm -rf interchaintest/coverage/Test_ChainStream_ConnectsAndReceivesEvents
	cd interchaintest && go test -timeout 30m -v -run Test_ChainStream_ConnectsAndReceivesEvents .
	./scripts/coverage-html.sh interchaintest/coverage/Test_ChainStream_ConnectsAndReceivesEvents

ictest-chainstream-websocket: rm-testcache
	rm -rf interchaintest/coverage/Test_ChainStreamWebsocket_ConnectsAndReceivesEvents
	cd interchaintest && go test -timeout 30m -v -run Test_ChainStreamWebsocket_ConnectsAndReceivesEvents .
	./scripts/coverage-html.sh interchaintest/coverage/Test_ChainStreamWebsocket_ConnectsAndReceivesEvents

ictest-downtime-detector: rm-testcache
	rm -rf interchaintest/coverage/TestDowntimeDetector
	cd interchaintest && go test -timeout 30m -v -run TestDowntimeDetector .
	./scripts/coverage-html.sh interchaintest/coverage/TestDowntimeDetector

ictest-validator-jailed: rm-testcache
	rm -rf interchaintest/coverage/Test_ValidatorJailedEvent
	cd interchaintest && go test -timeout 30m -v -run Test_ValidatorJailedEvent .
	./scripts/coverage-html.sh interchaintest/coverage/Test_ValidatorJailedEvent

ictest-wasm-fees-to-auction: rm-testcache
	rm -rf interchaintest/coverage/Test_WasmFeesGoToAuctionModule
	cd interchaintest && go test -timeout 30m -v -run Test_WasmFeesGoToAuctionModule .
	./scripts/coverage-html.sh interchaintest/coverage/Test_WasmFeesGoToAuctionModule

ictest-circle: rm-testcache
	rm -rf interchaintest/coverage/Test_Circle
	cd interchaintest && go test -timeout 30m -v -run Test_Circle .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Circle

ictest-chainlink-data-streams: rm-testcache
	rm -rf interchaintest/coverage/TestChainlinkDataStreamsReports
	cd interchaintest && go test -timeout 30m -v -run TestChainlinkDataStreamsReports .
	./scripts/coverage-html.sh interchaintest/coverage/TestChainlinkDataStreamsReports

ictest-ante-multisig: rm-testcache
	rm -rf interchaintest/coverage/Test_Ante_MultisigPubkeySignatures
	cd interchaintest && go test -timeout 30m -v -run Test_Ante_MultisigPubkeySignatures .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Ante_MultisigPubkeySignatures

ictest-peggy-bad-signature-replay: rm-testcache
	rm -rf interchaintest/coverage/Test_PeggyBadSignatureEvidenceMalleabilityReplay
	cd interchaintest && go test -timeout 30m -v -run Test_PeggyBadSignatureEvidenceMalleabilityReplay .
	./scripts/coverage-html.sh interchaintest/coverage/Test_PeggyBadSignatureEvidenceMalleabilityReplay

ictest-peggy-confirm-batch-unbonded: rm-testcache
	rm -rf interchaintest/coverage/Test_PeggyConfirmBatch_RejectUnbonded
	cd interchaintest && go test -timeout 30m -v -run Test_PeggyConfirmBatch_RejectUnbonded .
	./scripts/coverage-html.sh interchaintest/coverage/Test_PeggyConfirmBatch_RejectUnbonded

ictest-peggo-unbonded-valset-confirm-test: rm-testcache
	rm -rf interchaintest/coverage/Test_Peggo_UnbondedValidatorCannotSubmitValsetConfirm
	cd interchaintest && go test -timeout 30m -v -run Test_Peggo_UnbondedValidatorCannotSubmitValsetConfirm .
	./scripts/coverage-html.sh interchaintest/coverage/Test_Peggo_UnbondedValidatorCannotSubmitValsetConfirm

ictest-peggy-valset-slashing-rejoin: rm-testcache
	rm -rf interchaintest/coverage/Test_PeggyValsetSlashingAfterValidatorRejoin
	cd interchaintest && go test -timeout 30m -v -run Test_PeggyValsetSlashingAfterValidatorRejoin .
	./scripts/coverage-html.sh interchaintest/coverage/Test_PeggyValsetSlashingAfterValidatorRejoin

ictest-auction-chain-halt-protection: rm-testcache
	rm -rf interchaintest/coverage/Test_AuctionChainHaltProtection
	cd interchaintest && go test -timeout 30m -v -run Test_AuctionChainHaltProtection .
	./scripts/coverage-html.sh interchaintest/coverage/Test_AuctionChainHaltProtection

ictest-staking-delegation-receivers: rm-testcache
	rm -rf interchaintest/coverage/TestSetDelegationTransferReceivers
	cd interchaintest && go test -timeout 30m -v -run TestSetDelegationTransferReceivers .
	./scripts/coverage-html.sh interchaintest/coverage/TestSetDelegationTransferReceivers

.PHONY: rm-testcache rm-ic-coverage
.PHONY: ictest-all ictest-basic ictest-upgrade ictest-ibchooks ictest-cardano-ibc-client ictest-cardano-full-token-swap ictest-permissions-wasm-hook ictest-pfm ictest-lanes
.PHONY: ictest-fixed-gas ictest-fixed-gas-regression ictest-fixed-gas-cross-margin ictest-peggo ictest-peggo-ibc ictest-peggo-rate-limit ictest-evm ictest-circle
.PHONY: ictest-downtime-detector ictest-chainstream ictest-chainstream-websocket ictest-validator-jailed ictest-wasm-fees-to-auction ictest-chainlink-data-streams
.PHONY: ictest-ante-multisig ictest-peggy-bad-signature-replay ictest-peggo-unbonded-valset-confirm-test ictest-peggy-valset-slashing-rejoin ictest-peggy-confirm-batch-unbonded ictest-auction-chain-halt-protection ictest-oracle-morpho ictest-staking-delegation-receivers

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

.PHONY: peggo-wrappers

peggo-wrappers:
	@echo "Generating wrappers from Peggy contracts..."
	./scripts/peggo_wrappers.sh
