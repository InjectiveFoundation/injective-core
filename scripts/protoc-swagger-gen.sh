#!/usr/bin/env bash

set -eo pipefail

SWAGGER_TMP_DIR=tmp-swagger-gen
SWAGGER_BUILD_DIR=tmp-swagger-build
COMETBFT_VERSION_TAG=v1.0.1-inj.2
COSMOS_SDK_VERSION_TAG=v0.50.13-evm-comet1-inj.3
IBC_APPS_VERSION_BRANCH=release/v8-inj
IBC_GO_VERSION_TAG=v8.7.0-evm-comet1-inj
WASMD_VERSION_TAG=v0.53.2-evm-comet1-inj
HYPERLANE_COSMOS_VERSION_TAG=v1.0.0-evm-comet1-inj.1
rm -fr $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR
mkdir -p $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR

cd $SWAGGER_BUILD_DIR
mkdir -p proto
printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/InjectiveLabs/injective-core\n" > proto/buf.yaml
cp ../proto/buf.gen.swagger.yaml proto/buf.gen.swagger.yaml
cp -r ../proto/injective proto/
cp -r ../proto/osmosis proto/

# download third_party API definitions directly from git repositories
buf export https://github.com/InjectiveLabs/cosmos-sdk.git#tag=$COSMOS_SDK_VERSION_TAG --output=./third_party
buf export https://github.com/InjectiveLabs/ibc-go.git#tag=$IBC_GO_VERSION_TAG --exclude-imports --output=./third_party
buf export https://github.com/InjectiveLabs/wasmd.git#tag=$WASMD_VERSION_TAG --exclude-imports --output=./third_party
buf export https://github.com/InjectiveLabs/cometbft.git#tag=$COMETBFT_VERSION_TAG --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ics23.git --exclude-imports --output=./third_party
buf export https://github.com/InjectiveLabs/ibc-apps.git#branch=$IBC_APPS_VERSION_BRANCH --exclude-imports --output=./third_party --path=middleware/packet-forward-middleware/proto && mv ./third_party/middleware/packet-forward-middleware/proto/packetforward ./third_party
buf export https://github.com/InjectiveLabs/hyperlane-cosmos.git#tag=$HYPERLANE_COSMOS_VERSION_TAG,subdir=proto --exclude-imports --output=./third_party

echo "Generating swagger files"
proto_dirs=$(find ./proto ./third_party -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  if [[ ! -z "$query_file" ]]; then
    echo generating $query_file
    buf generate --template proto/buf.gen.swagger.yaml $query_file
  fi
done

echo "Generated swagger files"

cd ..
echo "Combining swagger files"

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths false --includeDefinitions true

echo "Cleaning up"

# clean swagger files
rm -rf $SWAGGER_TMP_DIR $SWAGGER_BUILD_DIR