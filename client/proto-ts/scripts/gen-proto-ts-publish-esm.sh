#!/usr/bin/env bash

## Publish core-proto-ts
cd ./client/proto-ts/gen/esm || exit

v=$(npm view @injectivelabs/core-proto-ts version)
echo "current package version: $v"

v1="${v%.*}.$((${v##*.}+1))"
echo "new package version: $v1"

npm version $v1
npm publish
