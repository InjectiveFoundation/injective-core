# Wasm Hook contract example

Serves as a wasm hook for permissions module.

It can act as a denom authority (create, mint and burn tokenfactory tokens), also as a wasm hook to implement smart restriction checks.

This contract example supports `ExecuteMsgs` msg, which will execute any chain messages on behalf of the contract, upon owner request. This way the project owners can instantiate the contract, and then command it to create new denom, to mint or burn tokens and to distribute them to users, or automate this actions to become triggered by user actions.

Wasm hook functionality is implemented via `SendRestriction` query, to which all the send params are passed: `fromAddress`, `toAddress`, `action` and `amounts`. It should return error in case action is prohibited for those addresses and amounts, otherwise it should return new `toAddress`. Most of the time this newAddress is the same as the `toAddress` which was passed to the query.

## Build

```sh
docker run --rm -v "$(pwd)":/code -v "$HOME/.cargo/git":/usr/local/cargo/git \
--mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target \
--mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
cosmwasm/rust-optimizer-arm64:0.12.13
```
