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

DOCKER=docker
protoVer=0.12.1
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto: proto-format proto-gen proto-swagger-gen

proto-gen:
	@$(protoImage) sh ./scripts/protocgen.sh

proto-swagger-gen:
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json ./proto

proto-check-breaking:
	@$(protoImage) buf breaking --against-input '.git#branch=main'

proto-ts:
	@$(protoImage) sh ./scripts/protoc-gen-ts.sh

publish-ts:
	@./client/proto-ts/scripts/gen-proto-ts-publish.sh

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
.PHONY: update-swagger-docs