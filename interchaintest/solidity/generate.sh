!/bin/sh -e

solc --combined-json abi,bin Counter.sol > Counter.json
abigen --combined-json Counter.json --pkg contracts --type Counter --out Counter.go
rm Counter.json

exit 0
