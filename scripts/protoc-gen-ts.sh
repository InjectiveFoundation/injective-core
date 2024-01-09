#!/usr/bin/env bash
set -eo pipefail
echo "Generating TS proto code"

rm -rf client/proto-ts/gen

TS_BUILD_DIR=tmp-ts-build
rm -fr $TS_BUILD_DIR && mkdir -p $TS_BUILD_DIR && cd $TS_BUILD_DIR
mkdir -p proto

printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/InjectiveLabs/injective-core\n" > proto/buf.yaml
cp ../proto/buf.gen.ts.yaml proto/buf.gen.ts.yaml
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

rm -rf ./cosmos-sdk && rm -rf ./wasmd

proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  echo Generating $dir ...
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    buf generate --template proto/buf.gen.ts.yaml $file
  done
done

cd ..
rm -fr $TS_BUILD_DIR

# cd proto
# # download third_party API definitions
# buf export https://github.com/cosmos/ibc-go.git --output=./third_party
# buf export https://github.com/cosmos/cosmos-sdk.git --exclude-imports --output=./third_party
# buf export https://github.com/CosmWasm/wasmd.git --exclude-imports --output=./third_party

# # compile proto definitions
# proto_dirs=$(find ./injective ./third_party -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
# for dir in $proto_dirs; do
#   echo Generating $dir ...
#   for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
#       buf generate --template buf.gen.ts.yaml --include-imports $file
#   done
# done

# cd ..

########################################
####### POST GENERATION CLEANUP ####### 
########################################

## 1. Replace package with our own fork
search1="@improbable-eng/grpc-web"
replace1="@injectivelabs/grpc-web"

FILES=$( find ./client/proto-ts/gen -type f )

for file in $FILES
do  
  sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" $file
done

## 2. Replace extension type to ignore on compile time 
search1="getExtension():"
replace1="// @ts-ignore \n  getExtension():"
search2="setExtension("
replace2="// @ts-ignore \n  setExtension("

FILES=$( find ./client/proto-ts/gen -type f -name '*.d.ts' )

for file in $FILES
do  
  sed -ie "s/${search1//\//\\/}/${replace1//\//\\/}/g" $file
  sed -ie "s/${search2//\//\\/}/${replace2//\//\\/}/g" $file
done

## 3. Compile TypeScript for ESM package
cp ./client/proto-ts/stub/index.ts.template ./client/proto-ts/gen/proto/index.ts

### ESM 
cp ./client/proto-ts/stub/package.json.esm.template ./client/proto-ts/gen/proto/package.json
cp ./client/proto-ts/stub/tsconfig.json.esm.template ./client/proto-ts/gen/proto/tsconfig.json
npm --prefix ./client/proto-ts/gen/proto install 
npm --prefix ./client/proto-ts/gen/proto run gen
cp ./client/proto-ts/stub/package.json.esm.template ./client/proto-ts/gen/core-proto-ts/esm/package.json

### CJS 
cp ./client/proto-ts/stub/package.json.cjs.template ./client/proto-ts/gen/proto/package.json
cp ./client/proto-ts/stub/tsconfig.json.cjs.template ./client/proto-ts/gen/proto/tsconfig.json
npm --prefix ./client/proto-ts/gen/proto install 
npm --prefix ./client/proto-ts/gen/proto run gen
cp ./client/proto-ts/stub/package.json.cjs.template ./client/proto-ts/gen/core-proto-ts/cjs/package.json

## 4. Setup proper package.json for both chain-api and core-proto-ts packages
cp ./client/proto-ts/stub/package.json.core-proto-ts.template ./client/proto-ts/gen/core-proto-ts/package.json
mkdir -p ./client/proto-ts/gen/chain-api
cp ./client/proto-ts/stub/package.json.chain-api.template ./client/proto-ts/gen/chain-api/package.json

# 5. Clean up folders
rm -rf ./client/proto-ts/temp
rm -rf ./client/proto-ts/gen/proto
find ./client/proto-ts/gen -name "*.jse" -type f -delete
find ./client/proto-ts/gen -name "*.tse" -type f -delete
find ./client/proto-ts/gen -name "*.jsone" -type f -delete