#!/usr/bin/env bash

rm -rf modules
mkdir -p modules
mkdir -p modules/Core
mkdir -p modules/Injective


for D in ../injective-chain/modules/*; do
  if [ -d "${D}" ]; then
    #rm -rf "modules/Injective/$(echo $D | awk -F/ '{print $NF}')"
    mkdir -p "modules/Injective/$(echo $D | awk -F/ '{print $NF}')" && cp -r $D/spec/* "$_"
  fi
done

# cat ../modules/README.md | sed 's/\.\/x/\/modules/g' | sed 's/spec\/README.md//g'

# Include the specs from Cosmos SDK
git clone https://github.com/cosmos/cosmos-sdk.git
mv cosmos-sdk/x/auth/spec/ ./modules/Core/auth
mv cosmos-sdk/x/bank/spec/ ./modules/Core/bank
mv cosmos-sdk/x/crisis/spec/ ./modules/Core/crisis
mv cosmos-sdk/x/distribution/spec/ ./modules/Core/distribution
mv cosmos-sdk/x/evidence/spec/ ./modules/Core/evidence
mv cosmos-sdk/x/gov/spec/ ./modules/Core/gov
mv cosmos-sdk/x/slashing/spec/ ./modules/Core/slashing
mv cosmos-sdk/x/staking/spec/ ./modules/Core/staking
mv cosmos-sdk/x/upgrade/spec/ ./modules/Core/upgrade
rm -rf cosmos-sdk
