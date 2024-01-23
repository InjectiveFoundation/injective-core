#!/usr/bin/env bash

set -eo pipefail

SWAGGER_TMP_DIR=tmp-swagger-gen
SWAGGER_BUILD_DIR=tmp-swagger-build
rm -fr $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR
mkdir -p $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR

cd $SWAGGER_BUILD_DIR
mkdir -p proto
printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/InjectiveLabs/injective-core\n" > proto/buf.yaml
cp ../proto/buf.gen.swagger.yaml proto/buf.gen.swagger.yaml
cp -r ../proto/injective proto/

# download third_party API definitions
git clone https://github.com/InjectiveLabs/cosmos-sdk.git -b v0.47.3-inj-9 --depth 1 --single-branch
git clone https://github.com/InjectiveLabs/wasmd -b v0.45.0-inj --depth 1 --single-branch

buf export ./cosmos-sdk --output=third_party
buf export ./wasmd --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ibc-go.git --exclude-imports --output=third_party
buf export https://github.com/tendermint/tendermint.git --exclude-imports --output=third_party
buf export https://github.com/cosmos/ics23.git --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ibc-apps.git --exclude-imports --output=./third_party --path=middleware/packet-forward-middleware/proto && mv ./third_party/middleware/packet-forward-middleware/proto/packetforward ./third_party

proto_dirs=$(find ./proto ./third_party -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  if [[ ! -z "$query_file" ]]; then
    echo generating $query_file
    buf generate --template proto/buf.gen.swagger.yaml $query_file
  fi
done

cd ..
# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# clean swagger files
rm -rf $SWAGGER_TMP_DIR $SWAGGER_BUILD_DIR