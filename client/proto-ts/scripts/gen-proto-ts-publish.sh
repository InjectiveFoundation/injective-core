#!/usr/bin/env bash

## Publish chain-api
cd ./client/proto-ts/gen/chain-api || exit

v=$(npm view @injectivelabs/chain-api version)
echo "current package version: $v"

v1="${v%.*}.$((${v##*.}+1))"
echo "new package version: $v1"

npm version $v1
npm publish

## Publish core-proto-ts
cd ./../core-proto-ts || exit

v=$(npm view @injectivelabs/core-proto-ts version)
echo "current package version: $v"

v1="${v%.*}.$((${v##*.}+1))"
echo "new package version: $v1"

npm version $v1
npm publish
