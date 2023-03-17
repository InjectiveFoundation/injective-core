#!/usr/bin/env bash

#rm -rf ./docs/develop/modules/Core
rm -rf ./docs/develop/modules/Injective
#mkdir ./docs/develop/modules/Core
mkdir ./docs/develop/modules/Injective


# Get Injective modules specs 

for D in ../injective-chain/modules/*; do
  if [ -d "${D}" ]; then
    #rm -rf "modules/Injective/$(echo $D | awk -F/ '{print $NF}')"
    mkdir -p "docs/develop/modules/Injective/$(echo $D | awk -F/ '{print $NF}')" && cp -r $D/spec/* "$_"
  fi
done

## Include the specs from latest Cosmos SDK
#git clone --depth 1 https://github.com/cosmos/cosmos-sdk.git
#
#for D in ./cosmos-sdk/x/*; do
#  if [ -d "${D}" ]; then
#    #rm -rf "modules/Injective/$(echo $D | awk -F/ '{print $NF}')"
#    mkdir -p "docs/develop/modules/Core/$(echo $D | awk -F/ '{print $NF}')" && cp -r $D/README.md "$_"
#  fi
#done
#
#rm -rf cosmos-sdk

# Include the specs from Cosmos SDK v0.45.11
#git clone --depth 1 --branch v0.45.11 https://github.com/cosmos/cosmos-sdk.git
#
#for D in ./cosmos-sdk/x/*; do
#  if [ -d "${D}" ]; then
#    #rm -rf "modules/Injective/$(echo $D | awk -F/ '{print $NF}')"
#    mkdir -p "docs/develop/modules/Core/$(echo $D | awk -F/ '{print $NF}')" && cp -r $D/spec/* "$_"
#    perl -i -pe 's/simd/injectived/g' docs/develop/modules/Core/$(echo $D | awk -F/ '{print $NF}')/*
#  fi
#done

#rm -rf cosmos-sdk