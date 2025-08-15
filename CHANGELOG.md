<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry is required to include a tag and
the Github PR reference in the following format:

* (<tag>) \#<pr-number> message

The tag should consist of where the change is being made ex. (exchange), (iavl), (rpc)
The PR numbers must be later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality and performance improvements.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes, except security related.
"Security" for security related changes and exploit fixes. NOT EXPORTED in auto-publishing process.
"API Breaking" for breaking Protobuf, gRPC and REST routes and types used by end-users.
"CLI Breaking" for breaking CLI commands.
Ref: https://keepachangelog.com/en/1.1.0/
-->

# Changelog

## [Unreleased]

## [v1.16.3]() todo link and date

### Bug Fixes

- (exchange)  Fixed historical v1 Exchange queries for pre v1.16 blocks

## [v1.16.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.16.0) - 2025-07-24

### Bug Fixes

- (exchange)  Fixed deadlock on transient store iterator mutex not being released on panic (out-of-gas).
- (evm)  Fixed EVM nonce increment for any type of EVM txns when tx contains multiple msgs.
- (exchange)  Fixed CLI commands to support `ExpirationBlock` while maintaining backwards compatibility.
- (exchange)  Fixed propogation of AdminInfo inputs to governance launches for Perpetuals and Expiry Futures markets.
- (peggo)  On failure, `Relayer` loop attempts to submit subsequent batch.
- (peggy)  Added a fix for Peggy.sol contract when paying out fees to the relayer who submitted the batch.
- (swagger)  Updated swagger config.json file to include all Cosmos SDK modules.
- (exchange)  Fixed incorrect emptiness check for conditional orderbooks.
- (exchange)  Fixed incorrect max derivative order value usage.
- (wasm)  Fixed issue in few wasm queries (human readable format).
- (exchange)  Fixed boundary constraints for IMR in `PerpetualMarketLaunchProposal` and `ExpiryFuturesMarketLaunchProposal`.

### Features

- (evm)  Native EVM support
- (ante)  Added support for Injective EVM Mainnet and Testnet in EIP712 Tx (chainID 1776 and 1439)
- (evm)  Added denom creation fee for STR erc20 denoms.
- (cmd)  Devnetify existing state via CLI `bootstrap-devnet` command.
- (chain-stream)  Added the gas price to the v2 chain stream response
- (exchange)  Refactoring of Exchange module to use human-readable values in all places except for deposits.
- (exchange)  Added GTB (Good-Til-Block) limit orders
- (exchange)  Added new reduce margin ratio for derivative markets
- (exchange)  Added new `EventTriggerConditionalMarketOrderFailed` and `EventTriggerConditionalLimitOrderFailed` events when a conditional order fails to execute after being triggered.

### Improvements

- (cometbft)  Upgraded CometBFT to v1.0.1
- (cosmos-sdk)  Updated to Cosmos SDK v0.50.13
- (peggo) Moved InjectiveLabs/peggo to injective-core repo.
- (peggo)  Expose loop durations through .env vars. Clients should not change the default values.
- (peggo)  Removed sdk-go dependency.
- (evm)  Migrated precompiles bindings generation to forge
- (cmd)  Added support for batching of multiple raw evm txns in `tx evm raw` CLI command.
- (evm)  Upgraded go-ethereum to v1.15.11
- (evm)  Disable unused gas refunds for MsgEthereumTx
- (cmd)  Removed rosetta dependency.

## [v1.15.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.15.0) - 2025-04-17

### Bug Fixes

- (wasmx)  Fixed wasmx authz ExecuteCompat authorization to work properly when MaxCalls > 1 filter is applied.

### Features

- (txfees)  Dynamic transaction fees with EIP-1559 style fee market. The implementation is based on the [Osmosis implementation](https://github.com/osmosis-labs/osmosis/tree/main/x/txfees).
- (cmd)  Add `--log-color` bool flag support to disable coloring of log lines, disable usage print on errors.
- (exchange)  Introduce fixed-gas consumption for certain exchange Msg types.
- (abci/block-sdk)  Added app-level mempool prioritization.
- (exchange)  CLI command for MsgWithdraw, MsgExternalTransfer

## [v1.14.1](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.1) - 2025-02-28

## [v1.14.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.0) - 2025-02-14

### Features

- (api)  Add Stork support in chain stream.
- (exchange)  Introduce support for spot market decimals.
- (exchange)  Introduce support for derivative market decimals.
- (oracle)  Add coinbase-price-states to CLI oracle query.
- (permissions)  Add permissions module asset freezing and token factory admin burn.
- (wasmx)  Support Authz grants for wasmx/MsgExecuteContractCompat.

### Improvements

- (docs)  Improve Peggy documentation.
- (docs)  Update oracle governance proposals info.
- (docs)  Generate module errors documentation.
- (docs)  Correct hyperlinks in BeginBlocker and EndBlocker documentation.
- (exchange)  Add subaccount balance check in invariants validation.
- (exchange)  Add market funds isolation for old markets.
- (exchange)  Enforce min notional for quote denoms on instant launch.
- (infra)  Update Docker image to match Go toolchain, remove old Dockerfile.release.
- (wasm)  Bump wasmd to v0.53.2-inj-1.

### Bug Fixes

- (api)  Remove reference to packet forward query in Swagger.
- (api)  Fix chain stream event parsing.
- (docs)  Fix duplicate documentation directory issue.
- (exchange)  Fix proposal handler trading rewards test.
- (exchange)  Market funds isolation fixes.
- (exchange)  Use existing decimals if spot update params proposal lacks decimals.
- (exchange)  Add MsgReclaimLockedFunds back into codec.
- (exchange)  Prevent admins from bypassing whitelisted min notional.
- (exchange)  Remove quote denoms min notional.
- (infra)  Fix release process for MacOS.
- (ledger)  Properly generate Ledger sign bytes.

### CLI Breaking Changes

- (cli)  Fix arguments in set-denom-metadata command.

## Previous Releases

[CHANGELOG of previous versions](https://github.com/InjectiveFoundation/injective-core/blob/v1.8/CHANGELOG.md#v17---2022-08-27) (last entry 2022-08-27).
