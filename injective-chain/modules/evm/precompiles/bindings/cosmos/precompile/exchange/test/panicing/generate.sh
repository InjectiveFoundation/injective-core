#!/bin/sh -e

solc --combined-json abi,bin Panicing.sol > Panicing.json
abigen --combined-json Panicing.json --pkg panicing --type Panicing --out Panicing.abi.go
rm Panicing.json

exit 0
