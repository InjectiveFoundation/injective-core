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

## [v1.16.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.16.0) - 2025-07-24

### Bug Fixes

- (evm) [#2250](https://github.com/InjectiveLabs/injective-core/pull/2250) Fixed EVM nonce increment for any type of EVM txns when tx contains multiple msgs.
- (exchange) [#2197](https://github.com/InjectiveLabs/injective-core/pull/2197) Fixed CLI commands to support `ExpirationBlock` while maintaining backwards compatibility.
- (exchange) [#2195](https://github.com/InjectiveLabs/injective-core/pull/2195) Fixed propogation of AdminInfo inputs to governance launches for Perpetuals and Expiry Futures markets.
- (peggo) [#2216](https://github.com/InjectiveLabs/injective-core/pull/2216) On failure, `Relayer` loop attempts to submit subsequent batch.
- (peggy) [#2186](https://github.com/InjectiveLabs/injective-core/pull/2186) Added a fix for Peggy.sol contract when paying out fees to the relayer who submitted the batch.
- (swagger) [#2242](https://github.com/InjectiveLabs/injective-core/pull/2242) Updated swagger config.json file to include all Cosmos SDK modules.
- (exchange) [#2246](https://github.com/InjectiveLabs/injective-core/pull/2246) Fixed incorrect emptiness check for conditional orderbooks.
- (exchange) [#2254](https://github.com/InjectiveLabs/injective-core/pull/2254) Fixed incorrect max derivative order value usage.
- (wasm) [#2260](https://github.com/InjectiveLabs/injective-core/pull/2260) Fixed issue in few wasm queries (human readable format).
- (exchange) [#2306](https://github.com/InjectiveLabs/injective-core/pull/2306) Fixed boundary constraints for IMR in `PerpetualMarketLaunchProposal` and `ExpiryFuturesMarketLaunchProposal`.

### Features

- (evm) [#2099](https://github.com/InjectiveLabs/injective-core/pull/2099) Native EVM support
- (ante) [#2275](https://github.com/InjectiveLabs/injective-core/pull/2275) Added support for Injective EVM Mainnet and Testnet in EIP712 Tx (chainID 1776 and 1439)
- (evm) [#2269](https://github.com/InjectiveLabs/injective-core/pull/2269) Added denom creation fee for STR erc20 denoms.
- (cmd) [#2143](https://github.com/InjectiveLabs/injective-core/pull/2143) Devnetify existing state via CLI `bootstrap-devnet` command.
- (chain-stream) [#2229](https://github.com/InjectiveLabs/injective-core/pull/2229) Added the gas price to the v2 chain stream response
- (exchange) [#2100](https://github.com/InjectiveLabs/injective-core/pull/2100) Refactoring of Exchange module to use human-readable values in all places except for deposits.
- (exchange) [#2175](https://github.com/InjectiveLabs/injective-core/pull/2175) Added GTB (Good-Til-Block) limit orders
- (exchange) [#2192](https://github.com/InjectiveLabs/injective-core/pull/2192) Added new reduce margin ratio for derivative markets
- (exchange) [#2187](https://github.com/InjectiveLabs/injective-core/pull/2187) Added new `EventTriggerConditionalMarketOrderFailed` and `EventTriggerConditionalLimitOrderFailed` events when a conditional order fails to execute after being triggered.

### Improvements

- (cometbft) [#2215](https://github.com/InjectiveLabs/injective-core/pull/2215) Upgraded CometBFT to v1.0.1
- (cosmos-sdk) [#2183](https://github.com/InjectiveLabs/injective-core/pull/2183) Updated to Cosmos SDK v0.50.13
- (peggo) Moved InjectiveLabs/peggo to injective-core repo.
- (peggo) [#2209](https://github.com/InjectiveLabs/injective-core/pull/2209) Expose loop durations through .env vars. Clients should not change the default values.
- (peggo) [#2219](https://github.com/InjectiveLabs/injective-core/pull/2219) Removed sdk-go dependency.
- (evm) [#2238](https://github.com/InjectiveLabs/injective-core/pull/2238) Migrated precompiles bindings generation to forge
- (cmd) [#2250](https://github.com/InjectiveLabs/injective-core/pull/2250) Added support for batching of multiple raw evm txns in `tx evm raw` CLI command.
- (evm) [#2241](https://github.com/InjectiveLabs/injective-core/pull/2241) Upgraded go-ethereum to v1.15.11
- (evm) [#2211](https://github.com/InjectiveLabs/injective-core/pull/2211) Disable unused gas refunds for MsgEthereumTx
- (cmd) [#2207](https://github.com/InjectiveLabs/injective-core/pull/2207) Removed rosetta dependency.

## [v1.15.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.15.0) - 2025-04-17

### Bug Fixes

- (wasmx) [#2136](https://github.com/InjectiveLabs/injective-core/pull/2136) Fixed wasmx authz ExecuteCompat authorization to work properly when MaxCalls > 1 filter is applied.

### Features

- (txfees) [#4266](https://github.com/InjectiveLabs/injective-core/pull/4266) Dynamic transaction fees with EIP-1559 style fee market. The implementation is based on the [Osmosis implementation](https://github.com/osmosis-labs/osmosis/tree/main/x/txfees).
- (cmd) [#2124](https://github.com/InjectiveLabs/injective-core/pull/2124) Add `--log-color` bool flag support to disable coloring of log lines, disable usage print on errors.
- (exchange) [#2096](https://github.com/InjectiveLabs/injective-core/pull/2096) Introduce fixed-gas consumption for certain exchange Msg types.
- (abci/block-sdk) [#2106](https://github.com/InjectiveLabs/injective-core/pull/2106) Added app-level mempool prioritization.
- (exchange) [#2160](https://github.com/InjectiveLabs/injective-core/pull/2160) CLI command for MsgWithdraw, MsgExternalTransfer

### Security

- (bank) [#2119](https://github.com/InjectiveLabs/injective-core/pull/2119) Add auction module address to bank blockedAddrs
- (permissions) [#2113](https://github.com/InjectiveLabs/injective-core/pull/2113) Do not return error when wasm hook is misbehaving.
- (ibc) [#2150](https://github.com/InjectiveLabs/injective-core/pull/2150) Bump ibc-go to v8.7.0-inj. Fixes GHSA-4wf3-5qj9-368v

## [v1.14.1](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.1) - 2025-02-28

### Security

- (sdk/ibc) [#2141](https://github.com/InjectiveLabs/injective-core/pull/2141) Bump SDK, IBC versions (ASA-0024-0012, ASA-0024-0013, ASA-2025-004, ASA-2024-010, GHSA-6fgm-x6ff-w78f)

## [v1.14.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.0) - 2025-02-14

### Features

- (api) [#1924](https://github.com/InjectiveLabs/injective-core/pull/1924) Add Stork support in chain stream.
- (exchange) [#1830](https://github.com/InjectiveLabs/injective-core/pull/1830) Introduce support for spot market decimals.
- (exchange) [#1847](https://github.com/InjectiveLabs/injective-core/pull/1847) Introduce support for derivative market decimals.
- (oracle) [#1948](https://github.com/InjectiveLabs/injective-core/pull/1948) Add coinbase-price-states to CLI oracle query.
- (permissions) [#1965](https://github.com/InjectiveLabs/injective-core/pull/1965) Add permissions module asset freezing and token factory admin burn.
- (wasmx) [#2059](https://github.com/InjectiveLabs/injective-core/pull/2059) Support Authz grants for wasmx/MsgExecuteContractCompat.

### Improvements

- (docs) [#1815](https://github.com/InjectiveLabs/injective-core/pull/1815) Improve Peggy documentation.
- (docs) [#1994](https://github.com/InjectiveLabs/injective-core/pull/1994) Update oracle governance proposals info.
- (docs) [#2025](https://github.com/InjectiveLabs/injective-core/pull/2025) Generate module errors documentation.
- (docs) [#2031](https://github.com/InjectiveLabs/injective-core/pull/2031) Correct hyperlinks in BeginBlocker and EndBlocker documentation.
- (exchange) [#1949](https://github.com/InjectiveLabs/injective-core/pull/1949) Add subaccount balance check in invariants validation.
- (exchange) [#2034](https://github.com/InjectiveLabs/injective-core/pull/2034) Add market funds isolation for old markets.
- (exchange) [#2049](https://github.com/InjectiveLabs/injective-core/pull/2049) Enforce min notional for quote denoms on instant launch.
- (infra) [#1957](https://github.com/InjectiveLabs/injective-core/pull/1957) Update Docker image to match Go toolchain, remove old Dockerfile.release.
- (wasm) [#2042](https://github.com/InjectiveLabs/injective-core/pull/2042) Bump wasmd to v0.53.2-inj-1.

### Bug Fixes

- (api) [#1912](https://github.com/InjectiveLabs/injective-core/pull/1912) Remove reference to packet forward query in Swagger.
- (api) [#2008](https://github.com/InjectiveLabs/injective-core/pull/2008) Fix chain stream event parsing.
- (docs) [#1913](https://github.com/InjectiveLabs/injective-core/pull/1913) Fix duplicate documentation directory issue.
- (exchange) [#2028](https://github.com/InjectiveLabs/injective-core/pull/2028) Fix proposal handler trading rewards test.
- (exchange) [#2035](https://github.com/InjectiveLabs/injective-core/pull/2035) Market funds isolation fixes.
- (exchange) [#2053](https://github.com/InjectiveLabs/injective-core/pull/2053) Use existing decimals if spot update params proposal lacks decimals.
- (exchange) [#2055](https://github.com/InjectiveLabs/injective-core/pull/2055) Add MsgReclaimLockedFunds back into codec.
- (exchange) [#2057](https://github.com/InjectiveLabs/injective-core/pull/2057) Prevent admins from bypassing whitelisted min notional.
- (exchange) [#2065](https://github.com/InjectiveLabs/injective-core/pull/2065) Remove quote denoms min notional.
- (infra) [#1904](https://github.com/InjectiveLabs/injective-core/pull/1904) Fix release process for MacOS.
- (ledger) [#1908](https://github.com/InjectiveLabs/injective-core/pull/1908) Properly generate Ledger sign bytes.

### CLI Breaking Changes

- (cli) [#1918](https://github.com/InjectiveLabs/injective-core/pull/1918) Fix arguments in set-denom-metadata command.

### Security

- (auction) [#2004](https://github.com/InjectiveLabs/injective-core/pull/2004) Add auction audit suggestions.
- (peggy) [#1986](https://github.com/InjectiveLabs/injective-core/pull/1986) Send bad deposits to a segregated wallet.
- (peggy) [#2012](https://github.com/InjectiveLabs/injective-core/pull/2012) Emit events for successful deposits/batches.
- (peggy) [#2033](https://github.com/InjectiveLabs/injective-core/pull/2033) Audit fixes.
- (peggy) [#2038](https://github.com/InjectiveLabs/injective-core/pull/2038) Enforce previous batch fee constraint.
- (wasm) [#2097](https://github.com/InjectiveLabs/injective-core/pull/2097) bump wasmvm v2.1.5 (CWA-2025-001, CWA-2025-002)
- (cometbft) [#2097](https://github.com/InjectiveLabs/injective-core/pull/2097) bump cometbft v0.38.17-inj-0 (advisory GHSA-r3r4-g7hq-pq4f)

## Previous Releases

[CHANGELOG of previous versions](https://github.com/InjectiveFoundation/injective-core/blob/v1.8/CHANGELOG.md#v17---2022-08-27) (last entry 2022-08-27).
