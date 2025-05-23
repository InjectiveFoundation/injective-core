#!/usr/bin/env bash

set -eo pipefail

SWAGGER_TMP_DIR=tmp-swagger-gen
SWAGGER_BUILD_DIR=tmp-swagger-build
COSMOS_SDK_VERSION_TAG=v0.50.13-evm-inj
IBC_GO_VERSION_TAG=v8.7.0-evm-inj
WASMD_VERSION_TAG=v0.53.2-evm-inj
rm -fr $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR
mkdir -p $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR

cd $SWAGGER_BUILD_DIR
mkdir -p proto
printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/InjectiveLabs/injective-core\n" > proto/buf.yaml
cp ../proto/buf.gen.swagger.yaml proto/buf.gen.swagger.yaml
cp -r ../proto/injective proto/
cp -r ../proto/osmosis proto/

# download third_party API definitions
git clone https://github.com/InjectiveLabs/cosmos-sdk.git -b $COSMOS_SDK_VERSION_TAG --depth 1 --single-branch
git clone https://github.com/InjectiveLabs/ibc-go.git -b $IBC_GO_VERSION_TAG --depth 1 --single-branch
git clone https://github.com/InjectiveLabs/wasmd.git -b $WASMD_VERSION_TAG --depth 1 --single-branch

buf export ./cosmos-sdk --output=./third_party
buf export ./ibc-go --exclude-imports --output=./third_party
buf export ./wasmd --exclude-imports --output=./third_party
buf export https://github.com/InjectiveLabs/cometbft.git --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ics23.git --exclude-imports --output=./third_party
buf export https://github.com/InjectiveLabs/ibc-apps.git --exclude-imports --output=./third_party --path=middleware/packet-forward-middleware/proto && mv ./third_party/middleware/packet-forward-middleware/proto/packetforward ./third_party

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

rm -rf ./cosmos-sdk && rm -rf ./ibc-go && rm -rf ./wasmd

cd ..
echo "Combining swagger files"

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

echo "Cleaning up"

# clean swagger files
rm -rf $SWAGGER_TMP_DIR $SWAGGER_BUILD_DIR