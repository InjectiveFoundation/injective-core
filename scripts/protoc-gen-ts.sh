#!/usr/bin/env bash
set -eo pipefail

TS_PROTO_TEMPLATE=proto/buf.gen.ts.yaml
TS_BUILD_DIR=tmp-ts-build
TS_OUTPUT_DIR=client/proto-ts/gen

# if hitting the BSD rate limit
buf registry login
counter=0

########################################
########## CODE GENERATION #############
########################################
echo "Generating TS proto code..."

rm -rf $TS_OUTPUT_DIR
rm -fr $TS_BUILD_DIR && mkdir -p $TS_BUILD_DIR && cd $TS_BUILD_DIR

mkdir -p proto
printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/InjectiveLabs/injective-core\n" > proto/buf.yaml
cp ../$TS_PROTO_TEMPLATE $TS_PROTO_TEMPLATE
cp -r ../proto/injective proto/

# download third_party API definitions
cosmos_sdk_branch=v0.50.x-inj
wasmd_branch=v0.50.x-inj

git clone https://github.com/InjectiveLabs/cosmos-sdk.git -b $cosmos_sdk_branch --depth 1 --single-branch > /dev/null
git clone https://github.com/InjectiveLabs/wasmd -b $wasmd_branch --depth 1 --single-branch > /dev/null

buf export ./cosmos-sdk --output=third_party
buf export ./wasmd --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ibc-go.git --exclude-imports --output=third_party
buf export https://github.com/cometbft/cometbft.git --exclude-imports --output=third_party
buf export https://github.com/cosmos/ics23.git --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ibc-apps.git --exclude-imports --output=./third_party --path=middleware/packet-forward-middleware/proto/packetforward/v1

rm -rf ./cosmos-sdk && rm -rf ./wasmd

# generate TS proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  echo Generating "$dir" ...
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    buf generate --template $TS_PROTO_TEMPLATE "$file"

    # if still hitting BSR rate limit, uncomment this
    ((counter++))
    if (( counter % 5 == 0 )); then
      sleep 8
    fi
  done
done

cd ..
rm -fr $TS_BUILD_DIR

#######################################
###### POST GENERATION CLEANUP #######
#######################################

echo "Compiling npm packages..."

## 1. Replace package with our own fork
search1="@improbable-eng/grpc-web"
replace1="@injectivelabs/grpc-web"

FILES=$( find ./$TS_OUTPUT_DIR -type f )

for file in $FILES
do
  sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" "$file"
done

## 2. Replace extension type to ignore on compile time
search1="getExtension():"
replace1="// @ts-ignore \n  getExtension():"
search2="setExtension("
replace2="// @ts-ignore \n  setExtension("

FILES=$( find ./$TS_OUTPUT_DIR -type f -name '*.d.ts' )

for file in $FILES
do
  sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" "$file"
  sed -ie "s/${search2//\//\\/}/${replace2//\//\\/}/g" "$file"
done

TS_STUB_DIR=client/proto-ts/stub
ESM_PKG_TEMPLATE=$TS_STUB_DIR/package.json.esm.template
ESM_CFG_TEMPLATE=$TS_STUB_DIR/tsconfig.json.esm.template
CJS_PKG_TEMPLATE=$TS_STUB_DIR/package.json.cjs.template
CJS_CFG_TEMPLATE=$TS_STUB_DIR/tsconfig.json.cjs.template

## 3. Compile TypeScript for ESM package
cp $TS_STUB_DIR/index.ts.template $TS_OUTPUT_DIR/proto/index.ts

### ESM
cp $ESM_PKG_TEMPLATE $TS_OUTPUT_DIR/proto/package.json
cp $ESM_CFG_TEMPLATE $TS_OUTPUT_DIR/proto/tsconfig.json
npm --prefix $TS_OUTPUT_DIR/proto install
npm --prefix $TS_OUTPUT_DIR/proto run gen
cp $ESM_PKG_TEMPLATE $TS_OUTPUT_DIR/core-proto-ts/esm/package.json

### CJS
cp $CJS_PKG_TEMPLATE $TS_OUTPUT_DIR/proto/package.json
cp $CJS_CFG_TEMPLATE $TS_OUTPUT_DIR/proto/tsconfig.json
npm --prefix $TS_OUTPUT_DIR/proto install
npm --prefix $TS_OUTPUT_DIR/proto run gen
cp $CJS_PKG_TEMPLATE $TS_OUTPUT_DIR/core-proto-ts/cjs/package.json

## 4. Setup proper package.json for both chain-api and core-proto-ts packages
cp $TS_STUB_DIR/package.json.core-proto-ts.template $TS_OUTPUT_DIR/core-proto-ts/package.json
mkdir -p $TS_OUTPUT_DIR/chain-api
cp $TS_STUB_DIR/package.json.chain-api.template $TS_OUTPUT_DIR/chain-api/package.json

# 5. Clean up folders
rm -rf client/proto-ts/temp
rm -rf $TS_OUTPUT_DIR/proto
find $TS_OUTPUT_DIR -name "*.jse" -type f -delete
find $TS_OUTPUT_DIR -name "*.tse" -type f -delete
find $TS_OUTPUT_DIR -name "*.jsone" -type f -delete

echo "Done!"