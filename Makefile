APP_VERSION = $(shell git describe --abbrev=0 --tags)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DATE = $(shell date -u "+%Y%m%d-%H%M")
COSMOS_VERSION_PKG = github.com/cosmos/cosmos-sdk/version
COSMOS_VERSION_NAME = injective
VERSION_PKG = github.com/InjectiveLabs/injective-core/version
IMAGE_NAME := injectivelabs/injective-core
LEDGER_ENABLED ?= true

ifeq ($(DO_COVERAGE),true)
coverage_flags = -coverpkg=`cat pkgs.txt`
else ifeq ($(DO_COVERAGE),yes)
coverage_flags = -coverpkg=`cat pkgs.txt`
else
coverage_flags =
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

pkgs.txt:
	go list \
		-f '{{if not .Standard}}{{.ImportPath}}{{end}}' \
		-deps ./cmd/... | \
	grep -E '^(cosmossdk\.io/|github\.com/bandprotocol/|github\.com/cometbft/|github\.com/cosmos/|github\.com/CosmWasm/|github\.com/ethereum/|github\.com/InjectiveLabs/)' | \
	paste -d "," -s - > pkgs.txt

install: init
install: export GOPROXY=direct
install: export VERSION_FLAGS="-X $(VERSION_PKG).AppVersion=$(APP_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)  -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=injectived -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install:
	go install -tags $(build_tags_comma_sep) $(BUILD_FLAGS) -ldflags $(VERSION_FLAGS) ./cmd/...

install-ci: pkgs.txt
install-ci: export GOPROXY=https://goproxy.injective.dev,direct 
install-ci: export VERSION_FLAGS="-X $(VERSION_PKG).AppVersion=$(APP_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)  -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(COSMOS_VERSION_PKG).Version=$(APP_VERSION) -X $(COSMOS_VERSION_PKG).Name=$(COSMOS_VERSION_NAME) -X $(COSMOS_VERSION_PKG).AppName=injectived -X $(COSMOS_VERSION_PKG).Commit=$(GIT_COMMIT)"
install-ci:
	go install -tags $(build_tags_comma_sep) $(BUILD_FLAGS) -ldflags $(VERSION_FLAGS) $(coverage_flags) ./cmd/...
	rm pkgs.txt

.PHONY: init install image push gen lint lint-last-commit test mock cover

mock: export GOPROXY=direct
mock: tests/mocks.go
	go install github.com/golang/mock/mockgen
	go generate ./tests/...

PKGS_TO_COVER := $(shell go list ./injective-chain/modules/exchange | paste -sd "," -)

deploy:
	./deploy_contracts.sh

###############################################################################
###                                   testing                               ###
###############################################################################
test: ictest-all test-unit

test-unit: export GOPROXY=direct
test-unit:
	go install github.com/onsi/ginkgo/ginkgo@latest
	ginkgo -r --race --randomizeSuites --randomizeAllSpecs --coverpkg=$(PKGS_TO_COVER) ./...

test-fuzz: # use old clang linker on macOS https://github.com/golang/go/issues/65169
	go test -fuzz FuzzTest ./injective-chain/modules/exchange/testexchange/fuzztesting -ldflags=-extldflags=-Wl,-ld_classic

test-erc20bridge:
	go test -v ./injective-chain/modules/erc20bridge/...

test-exchange:
	go test -v ./injective-chain/modules/exchange/...

test-rpc:
	MODE="rpc" go test -v ./tests/...

cover:
	go tool cover -html=tests/injective-chain/modules/exchange/exchange.coverprofile

.PHONY: test test-unit test-fuzz test-erc20bridge test-exchange test-rpc

# TODO: add runsim and benchmarking

###############################################################################
###                             e2e interchain test                         ###
###############################################################################

rm-testcache:
	go clean -testcache

rm-ic-coverage:
	rm -rf interchaintest/coverage

ictest-all: rm-testcache rm-ic-coverage
	cd interchaintest && go test -v -run ./...

ictest-basic: rm-testcache
	rm -rf interchaintest/coverage/TestBasicInjectiveStart
	cd interchaintest && go test -race -v -run TestBasicInjectiveStart .
	./scripts/coverage-html.sh interchaintest/coverage/TestBasicInjectiveStart

ictest-upgrade: rm-testcache
	rm -rf interchaintest/coverage/TestInjectiveUpgradeHandler
	cd interchaintest && go test -race -v -run TestInjectiveUpgradeHandler .
	./scripts/coverage-html.sh interchaintest/coverage/TestInjectiveUpgradeHandler

ictest-dynamic-fee: rm-testcache
	rm -rf interchaintest/coverage/TestDynamicFee
	cd interchaintest && go test -race -v -run Test_DynamicFee_FeeIncreases .
	./scripts/coverage-html.sh interchaintest/coverage/TestDynamicFee

ictest-ibchooks: rm-testcache
	rm -rf interchaintest/coverage/TestInjectiveIBCHooks
	cd interchaintest && go test -race -v -run TestInjectiveIBCHooks .
	./scripts/coverage-html.sh interchaintest/coverage/TestInjectiveIBCHooks

ictest-permissions-wasm-hook: rm-testcache
	rm -rf interchaintest/coverage/TestPermissionedDenomWasmHookCall
	cd interchaintest && go test -race -v -run TestPermissionedDenomWasmHookCall .
	./scripts/coverage-html.sh interchaintest/coverage/TestPermissionedDenomWasmHookCall

ictest-pfm: rm-testcache
	rm -rf interchaintest/coverage/TestPacketForwardMiddleware
	cd interchaintest && go test -race -v -run TestPacketForwardMiddleware .
	./scripts/coverage-html.sh interchaintest/coverage/TestPacketForwardMiddleware

ictest-lanes: rm-testcache
	rm -rf interchaintest/coverage/TestLanes
	cd interchaintest && go test -race -v -run MempoolLanes .
	./scripts/coverage-html.sh interchaintest/coverage/TestLanes

ictest-fixed-gas: rm-testcache
	rm -rf interchaintest/coverage/Test_FixedGas_HappyPath
	cd interchaintest && go test -race -v -run Test_FixedGas_HappyPath .
	./scripts/coverage-html.sh interchaintest/coverage/Test_FixedGas_HappyPath

ictest-fixed-gas-regression: rm-testcache
	rm -rf interchaintest/coverage/Test_FixedGas_Regression
	cd interchaintest && go test -race -v -run Test_FixedGas_Regression .
	./scripts/coverage-html.sh interchaintest/coverage/Test_FixedGas_Regression

.PHONY: rm-testcache rm-ic-coverage
.PHONY: ictest-all ictest-basic ictest-upgrade ictest-ibchooks ictest-permissions-wasm-hook ictest-pfm ictest-lanes
.PHONY: ictest-fixed-gas ictest-fixed-gas-regression

###############################################################################

lint: export GOPROXY=direct
lint:
	golangci-lint run --timeout=15m --new-from-rev=master

lint-last-commit: export GOPROXY=direct
lint-last-commit:
	golangci-lint run --timeout=15m -v --new-from-rev=HEAD~

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

gen-modules-errors-pages:
	@exec ./scripts/docs/generate_errors_docs.sh

.PHONY: update-swagger-docs
