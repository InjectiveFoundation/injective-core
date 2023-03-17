<a name="unreleased"></a>

## [Unreleased]

<a name="v"></a>

## [v] - 0001-01-01

<a name="v1.7"></a>

## [v1.7] - 2022-08-27

### Chore

- bump mainnet release workflow version
- remove random docs
- add release-staging workflow
- fix conflicts
- bump wasmd version
- update peggy blacklist with latest addresses
- re-gen
- fix logging format ([#878](https://github.com/InjectiveLabs/injective-core/issues/878))
- use info instead of error level for common error logs ([#876](https://github.com/InjectiveLabs/injective-core/issues/876))
- update upgrade tests
- remove unused ica arg for upgrade handler
- update to latest Mito contracts ([#844](https://github.com/InjectiveLabs/injective-core/issues/844))
- add node version name ([#846](https://github.com/InjectiveLabs/injective-core/issues/846))
- upgrade third party packages to latest versions ([#833](https://github.com/InjectiveLabs/injective-core/issues/833))
- exclude huge upgrade test dir from docker build
- fix peggy typo
- various cleanups to remove mention of legacy EVM stuff
- add version LDFLAGS ([#799](https://github.com/InjectiveLabs/injective-core/issues/799)) ([#821](https://github.com/InjectiveLabs/injective-core/issues/821))
- add version LDFLAGS ([#799](https://github.com/InjectiveLabs/injective-core/issues/799))

### Feat

- rename RegisterAsDMM to RewardsOptOut
- refactor of RO logic ([#886](https://github.com/InjectiveLabs/injective-core/issues/886))
- implement up to cancel amount query ([#864](https://github.com/InjectiveLabs/injective-core/issues/864))
- add upgrade handler for v1.7 upgrade
- Return error flags for batch order creation failures ([#857](https://github.com/InjectiveLabs/injective-core/issues/857))
- add subaccount orders query ([#859](https://github.com/InjectiveLabs/injective-core/issues/859))
- add indexer exchange, chronos k8s config
- implement transient spot order query ([#845](https://github.com/InjectiveLabs/injective-core/issues/845))
- Fixtures invariant checks

### Fix

- add missing event emit
- try emitting cacheCtx's event manager events
- emit events in cache context
- use same ctx for conditional market order trigger
- use same context for triggering conditional limit orders + event
- bad merge
- allow to deposit full amount to subaccount in Peggy ([#894](https://github.com/InjectiveLabs/injective-core/issues/894))
- use correct fee values in events ([#891](https://github.com/InjectiveLabs/injective-core/issues/891))
- use latest wasmd version with event attribute determinism fix ([#875](https://github.com/InjectiveLabs/injective-core/issues/875))
- add spot negative maker fee event bug fix ([#868](https://github.com/InjectiveLabs/injective-core/issues/868))
- delete denom decimals in upgrade handler
- use 0x prefixed hash representation in queries ([#861](https://github.com/InjectiveLabs/injective-core/issues/861))
- remove legacy Kovan chainID support
- allow zero fee discount value
- old registerUpgradeHandlers denomDecimals fix
- chain stresser fixes ([#823](https://github.com/InjectiveLabs/injective-core/issues/823))

### Refactor

- remove unnecessary separate event manager

### Test

- fuzz tests additions + improvements ([#789](https://github.com/InjectiveLabs/injective-core/issues/789))

### Update

- golang version

<a name="v1.6"></a>

## [v1.6] - 2022-07-01

### Added

- main file to run tests
- more cases for common exchange authz
- erious case for common authz, spots
- helper function, clean tests
- revoke cases
- revoke case
- happy cases for other derivative authz messages
- mock data for derivative grant test
- a scenario test for exchange authz
- spot authz test
- other cases for cancel + batchCancel
- more cases to deriv market + batch create deriv orders
- test cases for CreateDerivativeLimitOrderAuthz validateBasic()
- other case for batchUPdateOrderAuthz accept()
- casess BatchUpdateOrdersAuthz

### Chore

- add version LDFLAGS ([#799](https://github.com/InjectiveLabs/injective-core/issues/799)) ([#821](https://github.com/InjectiveLabs/injective-core/issues/821)) ([#822](https://github.com/InjectiveLabs/injective-core/issues/822))
- fix deactivate capitalization ([#815](https://github.com/InjectiveLabs/injective-core/issues/815))
- add force settlement subcommand to command handler
- add fee rates relation check in MsgInstantBinaryOptionsMarketLaunch
- Post only tests - PR fixes
- minor refactor
- cleanup prints and comments
- add minor code re-formatting
- add denom decimals to upgrade handler ([#779](https://github.com/InjectiveLabs/injective-core/issues/779))
- add copy proto file script to sdk-go script
- minor refactor to use GetDerivativeOrBinaryOptionsMarketWithMarkPrice
- return err
- - [@injectivelabs](https://github.com/injectivelabs)/chain-api[@1](https://github.com/1).8.0-rc2
- add provider type to exchange
- bump wasmd v0.27.0-inj, wasmvm v1.0.0
- rename wasm vault script
- increment subaccount nonce early ([#702](https://github.com/InjectiveLabs/injective-core/issues/702))
- bump tendermint protos to v0.34.19 ([#694](https://github.com/InjectiveLabs/injective-core/issues/694))
- bump cosmos-sdk v0.45.4 ([#693](https://github.com/InjectiveLabs/injective-core/issues/693))
- bump cosmos proto files ([#687](https://github.com/InjectiveLabs/injective-core/issues/687))
- Add minor refactors to Wasm Script ([#685](https://github.com/InjectiveLabs/injective-core/issues/685))
- remove auction keeper from exchange keeper ([#679](https://github.com/InjectiveLabs/injective-core/issues/679))
- bump ibc-go/v2 v2.2.0
- bump cosmos-sdk v0.45.2-inj
- add CLI support for PO order type in spot & deriv limit orders ([#653](https://github.com/InjectiveLabs/injective-core/issues/653))
- upd docker-compose
- import exchange types only once in test

### Docs

- add CHANGELOG file ([#800](https://github.com/InjectiveLabs/injective-core/issues/800))
- fixes
- update dev to latest ([#677](https://github.com/InjectiveLabs/injective-core/issues/677))

### Feat

- Add MsgRegisterAsDMM ([#662](https://github.com/InjectiveLabs/injective-core/issues/662))

### Feat

- add license for open sourcing ([#814](https://github.com/InjectiveLabs/injective-core/issues/814))
- add binary options launch/update and denom update in BatchExchangeModificationProposal
- add denom decimals ([#781](https://github.com/InjectiveLabs/injective-core/issues/781))
- Post only tests - added fixture tests for spot markets
- set architecture automatically
- add support for M1 build in DockerFile
- Negative maker fees - test for matched order
- Negative maker fees - fixed tests
- Negative maker fees - fixed tests
- Negative maker fees - added test for cancelling
- Negative maker fees - tests (wip)
- Negative maker fees - tests (wip)
- Negative maker fees - tests (wip)
- Negative maker fees - tests (wip)
- Account Msg Server batch processing - added binary options ([#750](https://github.com/InjectiveLabs/injective-core/issues/750))
- add new update sdk go script ([#744](https://github.com/InjectiveLabs/injective-core/issues/744))
- add inj-to-subaccount CLI method
- change midprice to midprice and tob query ([#736](https://github.com/InjectiveLabs/injective-core/issues/736))
- Added CLI methods for oracle providers ([#734](https://github.com/InjectiveLabs/injective-core/issues/734))
- Add new Mito grpc query to show all contracts ([#731](https://github.com/InjectiveLabs/injective-core/issues/731))
- Oracle Provider - unit tests, fixed some bugs ([#721](https://github.com/InjectiveLabs/injective-core/issues/721))
- improve spot matching algorithm ([#713](https://github.com/InjectiveLabs/injective-core/issues/713))
- add oracle wasm query capability (only volatility)
- provider oracle
- add spot vault test script
- add wasm spot orders query
- add mid price queries ([#703](https://github.com/InjectiveLabs/injective-core/issues/703))
- new synthetic derivative trade flow ([#683](https://github.com/InjectiveLabs/injective-core/issues/683))
- Update maker pools with MITO master functionality ([#680](https://github.com/InjectiveLabs/injective-core/issues/680))
- Cosmwasm integration ([#607](https://github.com/InjectiveLabs/injective-core/issues/607))
- add rollback capabilities

### Feature

- Use new FBA limit clearing price ([#646](https://github.com/InjectiveLabs/injective-core/issues/646))

### Fix

- release GH workflow
- add nil check in matching
- add ICA controller and host storekey + migration code ([#805](https://github.com/InjectiveLabs/injective-core/issues/805))
- make trading reward points based on volume, not fee contribution ([#802](https://github.com/InjectiveLabs/injective-core/issues/802))
- add missing coma
- allow negative maker fees in instant perp/binary/futures market launches ([#797](https://github.com/InjectiveLabs/injective-core/issues/797))
- denom decimals setup + unmarshalling fix ([#796](https://github.com/InjectiveLabs/injective-core/issues/796))
- binary options fixtures
- change UST to USTC ([#787](https://github.com/InjectiveLabs/injective-core/issues/787))
- use copy of value in closure
- binary options post-trade margining fix
- return correct margin needed
- change Kovan chainID to Goerli ([#727](https://github.com/InjectiveLabs/injective-core/issues/727))
- add Route to MsgBatchCancelBinaryOptionsOrders
- add denom decimals for fixing volatility precision loss + few other fixes ([#778](https://github.com/InjectiveLabs/injective-core/issues/778))
- add missing market order validation for netting ([#767](https://github.com/InjectiveLabs/injective-core/issues/767))
- use insurance fund in TEF settlements ([#761](https://github.com/InjectiveLabs/injective-core/issues/761))
- margin hold should only be based on positive fee rate part ([#758](https://github.com/InjectiveLabs/injective-core/issues/758))
- remove redundant vesting CLI command ([#754](https://github.com/InjectiveLabs/injective-core/issues/754))
- add spot wasm script fixes from Peiyun ([#745](https://github.com/InjectiveLabs/injective-core/issues/745))
- wasmx tx example ([#742](https://github.com/InjectiveLabs/injective-core/issues/742))
- panic if error is not nil
- add fixes for negative maker fee markets with fee discounts ([#730](https://github.com/InjectiveLabs/injective-core/issues/730))
- run peggy module endblocker before exchange module endblocker ([#726](https://github.com/InjectiveLabs/injective-core/issues/726))
- minor comments and error type
- remove redundant rosetta cmd
- nil dereference check, array length vs capacity fix
- exchange tests
- genesis import
- add back in release workflow
- k8s resource config typo, tune resoure usage, add debug mode
- pass parameters as nil if not used in cli
- use optional decimal flag helper
- dont pass default values in cli
- use InjectiveLabs/cosmos-sdk v0.45.0-inj-2 ([#636](https://github.com/InjectiveLabs/injective-core/issues/636))

### Test

- add binary options fixture invariance check

### Tmp

- re-enable kovan ([#804](https://github.com/InjectiveLabs/injective-core/issues/804))

<a name="v1.1.3-test"></a>

## [v1.1.3-test] - 2022-02-16

### Chore

- write key as hex id and put higher to avoid future potential collisions
- dont import same package twice
- cleanup raw json string manipulation
- add market making pool grpc query
- refactor wasm mm code
- nuke old code
- renames
- migrate wasm stuff from exchange to wasmx module
- register codec + separate WasmMsgServer
- add cosmwasm proto files and swagger
- bump CosmWasm/wasmd v0.23.0 + regen
- add cosmwasm proto files and swagger
- bump cosmos-sdk v0.45.0-injective ([#621](https://github.com/InjectiveLabs/injective-core/issues/621))
- add authz, tendermint query and params to swagger
- add authz, tendermint query and params to swagger
- add cosmwasm proto files and swagger
- nuke hi.json
- re-gen ([#603](https://github.com/InjectiveLabs/injective-core/issues/603))
- simplify hex string validation
- add logs
- format proto files
- fix tests
- refactor has duplicates check
- minor refactors
- add reward points update proposal to specs
- rename clearingRefund to clearingChargeOrRefund in spot
- use info instead of error for log
- re-gen
- add some clarifying comments
- bump ibc-go v2.0.2
- further concepts specs update
- restructure exchange specs
- only retrieve orders when required for param update
- bump ibc-go
- bring proposal specs up-to-date + other minor updates
- bump cosmos-sdk to v0.44.5
- add new SubaccountOrderMetadata grpc query
- add fee discount ttl to grpc query
- regen docs
- update exchange specs
- add comments to genesis proto
- fix invalid path to image
- use default address constant in tests everywhere
- remove duplicate liquidation check
- improve comments and naming
- add price validation checks for safety
- bump ibc-go

### Feat

- finish wasm querier integration with batch update response
- call mm contract from begin blocker
- add subscribe and redeem mm pool functions and testing script
- initial wasmx module setup
- add WIP market maker subscribe msg
- add more msgs and queries for wasm contracts
- add query plugin structure + exchange wasm interface
- toy demo of querying & executing CW contracts in BeginBlocker
- add WasmViewKeeper and WasmContractOpsKeeper to exchange keeper
- initial wasmd integration using cosmoscontracts/wasmd
- add more msgs and queries for wasm contracts
- add query plugin structure + exchange wasm interface
- toy demo of querying & executing CW contracts in BeginBlocker
- add WasmViewKeeper and WasmContractOpsKeeper to exchange keeper
- initial wasmd integration using cosmoscontracts/wasmd
- spot and deriv cancel all in batchUpdate
- partial implementation for BatchUpdateOrders
- finish ValidateBasic for MsgBatchUpdateOrdersResponse
- add fee recipient address to trade logs
- add reward points update proposal to cli
- add new trading reward points update proposal
- send distr module fees to auction or market fees to insurance fund
- fix redemptions, add EventUnderwrite, add validation in RequestRedemption
- add staking requirement to trade & earn
- emit more data for liquidations
- emit EventAuctionStart

### Feat

- Add fee discount tier stats GRPC query ([#626](https://github.com/InjectiveLabs/injective-core/issues/626))
- Implement vested trading rewards ([#610](https://github.com/InjectiveLabs/injective-core/issues/610))
- Add new grpc balance queries ([#601](https://github.com/InjectiveLabs/injective-core/issues/601))

### Fix

- fix the proof of concept market making integration
- add wasm module to begin, end block order
- correctly iterate over pending pools in grpc query ([#620](https://github.com/InjectiveLabs/injective-core/issues/620))
- use correct InjRewardStakedRequirementThreshold in migration
- add IBC antehandler ([#612](https://github.com/InjectiveLabs/injective-core/issues/612))
- emit correct trade type for trade event during settlement
- register TradingRewardPointsUpdateProposal properly in codec ([#602](https://github.com/InjectiveLabs/injective-core/issues/602))
- use correct transient spot limit orders inside transient store
- use correct order for cancel orders
- additional validation for MsgBatchUpdateOrders
- don't require subaccountID if not cancelling all
- add auction module balance check in fuzz
- use zero hash/bytes for TradeLogs from market settlement
- rebase refactor
- trading rewards points update test fix
- prevent points from being increased
- validate no duplicate accounts in proposal
- cancel all derivative orders during settlement, not just from positions
- refund margin in case where fill quantity is zero
- prevent repeated coins
- use correct distribution subaccountID
- Update TStoreKey
- use custom cosmos fork
- convert band prices to dec before division
- allow unspecified market status in query, CLI desc fixes
- allow unspecified market status in query, CLI desc fixes
- CLI command for deriv param update
- only set new account tier ttl for fee discounts if actually expired
- add additional validators for batch Msgs

### Fix

- Add fix for fee discount proposal bug ([#615](https://github.com/InjectiveLabs/injective-core/issues/615))
- Add some minor edge case liquidation fixes ([#614](https://github.com/InjectiveLabs/injective-core/issues/614))
- Add fixes for GRPC balance mismatch queries ([#609](https://github.com/InjectiveLabs/injective-core/issues/609))

### Refactor

- emit new claim event only once
- add BatchTimeout to EventOutgoingBatch
- always emit claim events regardless of attestation
- spot and deriv orders to create/cancel
- minor refactors
- rename amt to amount

### Test

- finish batch update tests ([#600](https://github.com/InjectiveLabs/injective-core/issues/600))
- add transient order tests for batch update tests
- add batch order cancellation tests
- add proper logging for cancelling orders in all markets test
- dry up order creation
- use correct sender and subaccount id
- add order creation for markets in batch test
- refactor marketID checks
- add BatchUpdateOrders stateless tests + IsHexHash helper
- initial test setup and basic test
- add reward points update proposal test
- re-introduce max load fuzz test after deadlock fix
- add test for edge case market order margin refund
- add test for staking requirement in trade & earn
- fix TEF tests
- add back fuzz test
- fix funding rates tests
- fix the fixture tests for new simapp
- add fee discount test with caching and TTL expiry edge case

### Wip

- add MsgBatchUpdateOrders skeleton

<a name="v1.1.2"></a>

## [v1.1.2] - 2021-11-12

### Chore

- don't introduce potentially consensus breaking change
- rename test function
- DRY up derivative order fee calculations
- minor style nit

### Feat

- add new SubaccountPositions grpc query

### Fix

- use correct clearing price for edge case with far off-priced limit orders
- refund correct amount for negative maker fee derivative markets upon order cancellations ([#544](https://github.com/InjectiveLabs/injective-core/issues/544))

### Test

- add fixture test for out of range clearing price

<a name="v1.1.1"></a>

## [v1.1.1] - 2021-11-08

### Build

- yarn

### Chore

- add OCR tests to CI
- remove accidentally added comment
- update github workflow go version
- add mark price to EventPerpetualMarketFundingUpdate
- bump go.mod golang version to 1.17
- fix oracle tests
- bump cosmos-sdk
- fix comments, closes https://github.com/InjectiveLabs/injective-core/issues/496
- Add minor reduce only cancellation and fuzz test refactors ([#489](https://github.com/InjectiveLabs/injective-core/issues/489))
- add liquitity mining reward distribution tests
- bump ibc-go
- remove old genesis dir

### Docs

- fix typo

### Feat

- allow for changing oracle params in deriv markets, nuke Derivat… ([#520](https://github.com/InjectiveLabs/injective-core/issues/520))
- BatchExchangeModificationProposal ([#519](https://github.com/InjectiveLabs/injective-core/issues/519))
- add events for fee discounts and trading rewards
- add inj-address-from-eth-address query method
- add order hash in create market order response
- initial negative derivative maker fee implementation ([#476](https://github.com/InjectiveLabs/injective-core/issues/476))
- return useful order info upon sending exchange Msg ([#468](https://github.com/InjectiveLabs/injective-core/issues/468))

### Feature

- Only check past fees paid when first period has passed ([#513](https://github.com/InjectiveLabs/injective-core/issues/513))
- Add liquidity mining scheduling system ([#500](https://github.com/InjectiveLabs/injective-core/issues/500))
- Send market launch fees to Community Spend Pool ([#486](https://github.com/InjectiveLabs/injective-core/issues/486))
- implement derivative transient order cancels ([#483](https://github.com/InjectiveLabs/injective-core/issues/483))

### Fix

- Set past trading fees for fee discounts correctly ([#527](https://github.com/InjectiveLabs/injective-core/issues/527))

### Fix

- allow derivative market launches with negative maker fee ([#539](https://github.com/InjectiveLabs/injective-core/issues/539))
- count negative maker fees with negative multiplier in spot ([#534](https://github.com/InjectiveLabs/injective-core/issues/534))
- add IsFirstCycle to genesis and refactor to TrueByte in keeper
- add missing set for market fee discount qualification upon launch
- set flag once first fee cycle is finished instead of using bucket timestamps
- cap insurance fund underwriting to 1T \* 1e18 for overflow protection ([#532](https://github.com/InjectiveLabs/injective-core/issues/532))
- show correct tier when first bucket period is not over
- FeeDiscountAccountInfo query
- FeeDiscountAccountInfo query
- FeeDiscountAccountInfo query
- use copy of value in closure
- add nil check in ExecuteDerivativeMarketOrderMatching
- all scripts should use externally available bins via $PATH, including protoc-gen-ts
- - [@injectivelabs](https://github.com/injectivelabs)/chain-api[@1](https://github.com/1).4.16
- - [@injectivelabs](https://github.com/injectivelabs)/chain-api[@1](https://github.com/1).4.15
- gen script + [@injectivelabs](https://github.com/injectivelabs)/chain-api[@1](https://github.com/1).4.14
- replaced new logo
- exchange CLI fix
- CLI tx NewSpotMarketUpdateParamsProposalTxCmd
- only update deposits for vanilla orders ([#463](https://github.com/InjectiveLabs/injective-core/issues/463))
- CumulativeFundingEntry fix ([#460](https://github.com/InjectiveLabs/injective-core/issues/460))
- add missing RegisterTendermintService
- missing legacy amino codec hooks

### OCR

- fix onchain config ([#535](https://github.com/InjectiveLabs/injective-core/issues/535))

### Refactor

- use sdk.AccAddress ([#524](https://github.com/InjectiveLabs/injective-core/issues/524))
- added favicon

### Test

- make market param updates valid for fuzz tests ([#543](https://github.com/InjectiveLabs/injective-core/issues/543))
- adapt fee discount tests to new fee cycle mechanism
- fix test for multiple funding epochs

<a name="v1.1.0"></a>

## [v1.1.0] - 2021-11-06

<a name="v1.0.7"></a>

## [v1.0.7] - 2021-08-30

<a name="v1.0.6"></a>

## [v1.0.6] - 2021-08-27

<a name="v1.0.5"></a>

## [v1.0.5] - 2021-07-30

<a name="v1.0.4"></a>

## [v1.0.4] - 2021-07-30

<a name="v1.0.3"></a>

## [v1.0.3] - 2021-07-30

### Chore

- update the CanaryV2Block height
- cleanup dead peggy key code
- test entire module instead of just keeper ([#422](https://github.com/InjectiveLabs/injective-core/issues/422))

### Fix

- use notional in funding payment calculation ([#433](https://github.com/InjectiveLabs/injective-core/issues/433))
- log level, closes https://github.com/InjectiveLabs/injective-core/issues/419 ([#425](https://github.com/InjectiveLabs/injective-core/issues/425))
- re-enable fuzz tests
- use correct Bech32 fee recipient while supporting legacy canary chain misimplementation ([#420](https://github.com/InjectiveLabs/injective-core/issues/420))

<a name="v1.0.2"></a>

## [v1.0.2] - 2021-07-13

<a name="v1.0.1"></a>

## [v1.0.1] - 2021-06-29

<a name="v1.0.0"></a>

## [v1.0.0] - 2021-06-27

<a name="v1.0"></a>

## v1.0 - 2021-06-27

### Ante

- update nonce check

### Evm

- implement Homestead / EIP155 fallback for externally singed txs

### Keys

- fix privkey derivation

### Peggy

- MsgSubmitBadSignatureEvidence - no signer [#366](https://github.com/InjectiveLabs/injective-core/issues/366)

[unreleased]: https://github.com/InjectiveLabs/injective-core/compare/v...HEAD
[v]: https://github.com/InjectiveLabs/injective-core/compare/v1.7...v
[v1.7]: https://github.com/InjectiveLabs/injective-core/compare/v1.6...v1.7
[v1.6]: https://github.com/InjectiveLabs/injective-core/compare/v1.1.3-test...v1.6
[v1.1.3-test]: https://github.com/InjectiveLabs/injective-core/compare/v1.1.2...v1.1.3-test
[v1.1.2]: https://github.com/InjectiveLabs/injective-core/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/InjectiveLabs/injective-core/compare/v1.1.0...v1.1.1
[v1.1.0]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.7...v1.1.0
[v1.0.7]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.6...v1.0.7
[v1.0.6]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.5...v1.0.6
[v1.0.5]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.4...v1.0.5
[v1.0.4]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/InjectiveLabs/injective-core/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/InjectiveLabs/injective-core/compare/v1.0...v1.0.0
