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

## [v1.14.1](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.1) - 2025-02-28

## [v1.14.0](https://github.com/InjectiveFoundation/injective-core/releases/tag/v1.14.0) - 2025-02-14

### Features

* (api) [#1924](https://github.com/InjectiveLabs/injective-core/pull/1924) Add Stork support in chain stream.
* (exchange) [#1830](https://github.com/InjectiveLabs/injective-core/pull/1830) Introduce support for spot market decimals.
* (exchange) [#1847](https://github.com/InjectiveLabs/injective-core/pull/1847) Introduce support for derivative market decimals.
* (oracle) [#1948](https://github.com/InjectiveLabs/injective-core/pull/1948) Add coinbase-price-states to CLI oracle query.
* (permissions) [#1965](https://github.com/InjectiveLabs/injective-core/pull/1965) Add permissions module asset freezing and token factory admin burn.
* (wasmx) [#2059](https://github.com/InjectiveLabs/injective-core/pull/2059) Support Authz grants for wasmx/MsgExecuteContractCompat.

### Improvements

* (docs) [#1815](https://github.com/InjectiveLabs/injective-core/pull/1815) Improve Peggy documentation.
* (docs) [#1994](https://github.com/InjectiveLabs/injective-core/pull/1994) Update oracle governance proposals info.
* (docs) [#2025](https://github.com/InjectiveLabs/injective-core/pull/2025) Generate module errors documentation.
* (docs) [#2031](https://github.com/InjectiveLabs/injective-core/pull/2031) Correct hyperlinks in BeginBlocker and EndBlocker documentation.
* (exchange) [#1949](https://github.com/InjectiveLabs/injective-core/pull/1949) Add subaccount balance check in invariants validation.
* (exchange) [#2034](https://github.com/InjectiveLabs/injective-core/pull/2034) Add market funds isolation for old markets.
* (exchange) [#2049](https://github.com/InjectiveLabs/injective-core/pull/2049) Enforce min notional for quote denoms on instant launch.
* (infra) [#1957](https://github.com/InjectiveLabs/injective-core/pull/1957) Update Docker image to match Go toolchain, remove old Dockerfile.release.
* (wasm) [#2042](https://github.com/InjectiveLabs/injective-core/pull/2042) Bump wasmd to v0.53.2-inj-1.

### Bug Fixes

* (api) [#1912](https://github.com/InjectiveLabs/injective-core/pull/1912) Remove reference to packet forward query in Swagger.
* (api) [#2008](https://github.com/InjectiveLabs/injective-core/pull/2008) Fix chain stream event parsing.
* (docs) [#1913](https://github.com/InjectiveLabs/injective-core/pull/1913) Fix duplicate documentation directory issue.
* (exchange) [#2028](https://github.com/InjectiveLabs/injective-core/pull/2028) Fix proposal handler trading rewards test.
* (exchange) [#2035](https://github.com/InjectiveLabs/injective-core/pull/2035) Market funds isolation fixes.
* (exchange) [#2053](https://github.com/InjectiveLabs/injective-core/pull/2053) Use existing decimals if spot update params proposal lacks decimals.
* (exchange) [#2055](https://github.com/InjectiveLabs/injective-core/pull/2055) Add MsgReclaimLockedFunds back into codec.
* (exchange) [#2057](https://github.com/InjectiveLabs/injective-core/pull/2057) Prevent admins from bypassing whitelisted min notional.
* (exchange) [#2065](https://github.com/InjectiveLabs/injective-core/pull/2065) Remove quote denoms min notional.
* (infra) [#1904](https://github.com/InjectiveLabs/injective-core/pull/1904) Fix release process for MacOS.
* (ledger) [#1908](https://github.com/InjectiveLabs/injective-core/pull/1908) Properly generate Ledger sign bytes.

### CLI Breaking Changes

* (cli) [#1918](https://github.com/InjectiveLabs/injective-core/pull/1918) Fix arguments in set-denom-metadata command.

## Previous Releases

[CHANGELOG of previous versions](https://github.com/InjectiveFoundation/injective-core/blob/v1.8/CHANGELOG.md#v17---2022-08-27) (last entry 2022-08-27).
