---
sidebar_position: 5
title: State Transitions
---

# State Transitions

This document describes the state transition operations pertaining to:

- Deposit into exchange module account
- Withdraw from exchange module account
- Instant spot market launch
- Instant perpetual market launch
- Instant expiry futures market launch
- Spot limit order creation
- Batch creation of spot limit orders
- Spot market order creation
- Cancel spot order
- Batch cancellation of spot order
- Derivative limit order creation
- Batch derivative limit order creation
- Derivative market order creation
- Cancel derivative order
- Batch cancellation of derivative orders
- Transfer between subaccounts
- Transfer to external account
- Liquidating a position
- Increasing position margin
- Spot market param update proposal
- Exchange enable proposal
- Spot market launch proposal
- Perpetual market launch proposal
- Expiry futures market launch proposal
- Derivative market param update proposal
- Trading rewards launch proposal
- Trading rewards update proposal
- Begin-blocker
- End-blocker

## Deposit into exchange module account

Deposit action is carried out by `MsgDeposit` which consists of `Sender`, `SubaccountId` and `Amount` fields.

**Note:** `SubaccountId` is optional and if it's not available, it's calculated dynamically from `Sender` address.

**Steps**

- Check that the denom specified in `msg.Amount` is a valid denom which exists in bank supply
- Send coins from individual account to `exchange` module account and if fail, just revert
- Get hash type of `subaccountID` from `msg.SubaccountId`, if it's zero subaccount, calculate dynamically from `msg.Sender` by using `SdkAddressToSubaccountID`
- Increment deposit amount for the `subaccountID` by `msg.Amount`
- Emit event for `EventSubaccountDeposit` with `msg.Sender`, `subaccountID` and `msg.Amount`

## Withdraw from exchange module account

Withdraw action is carried out by `MsgWithdraw` which consists of `Sender`, `SubaccountId` and `Amount` fields.

**Note:** The ownership of `msg.SubaccountId` by `msg.Sender` is validated on `msg.ValidateBasic` function.

**Steps**

- Get hash type of `subaccountID` from `msg.SubaccountId`
- Check the denom specified in `msg.Amount` is a valid denom which exists in bank supply
- Decrement withdraw amount from `subaccountID` by `msg.Amount`, if fail, revert
- Send coins from `exchange` module to `msg.Sender`
- Emit event for `EventSubaccountWithdraw` with `subaccountID`, `msg.Sender`, and `msg.Amount`

## Instant spot market launch

Instant spot market launch action is carried out by `MsgInstantSpotMarketLaunch` which consists of `Sender`, `Ticker`, `BaseDenom`, `QuoteDenom`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- Calculate `marketID` from `msg.BaseDenom` and `msg.QuoteDenom`
- Check if same market launch proposal exists by `marketID` and revert if already exists
- Launch spot market with `msg.Ticker`, `msg.BaseDenom`, `msg.QuoteDenom`, `msg.MinPriceTickSize`, `msg.MinQuantityTickSize` and revert if fail
- Send instant listing fee(params.SpotMarketInstantListingFee) from `msg.Sender` to `exchange` module account
- Lastly send the instant listing fee to the community spend pool

## Instant perpetual market launch

Instant perpetual market launch action is carried out by `MsgInstantPerpetualMarketLaunch` which consists of `Sender`, `Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleScaleFactor`, `OracleType`, `MakerFeeRate`, `TakerFeeRate`, `InitialMarginRatio`, `MaintenanceMarginRatio`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- Calculate `marketID` from `msg.Ticker`, `msg.QuoteDenom`, `msg.OracleBase`, `msg.OracleQuote` and `msg.OracleType`.
- Check if same market launch proposal exists by `marketID` and revert if already exists
- Send instant listing fee(params.DerivativeMarketInstantListingFee) from `msg.Sender` to `exchange` module account
- Launch perpetual market with required params on `msg` object and revert if fail
- Lastly send the instant listing fee to the community spend pool

## Instant expiry futures market launch

Instant expiry futures market launch action is carried out by `MsgInstantExpiryFuturesMarketLaunch` which consists of `Sender`, `Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleScaleFactor`, `OracleType`, `Expiry`, `MakerFeeRate`, `TakerFeeRate`, `InitialMarginRatio`, `MaintenanceMarginRatio`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- Calculate `marketID` from `msg.Ticker`, `msg.QuoteDenom`, `msg.OracleBase`, `msg.OracleQuote`, `msg.OracleType` and `msg.Expiry`.
- Check if same market launch proposal exists by `marketID` and revert if already exists
- Send instant listing fee(params.DerivativeMarketInstantListingFee) from `msg.Sender` to `exchange` module account
- Launch expiry futures market with required params on `msg` object and revert if fail
- Trigger `EventExpiryFuturesMarketUpdate` event with market info
- Lastly send the instant listing fee to the community spend pool

## Spot limit order creation

Spot limit order creation is carried out by `MsgCreateSpotLimitOrder` which consists of `Sender` and `Order`.

**Steps**

- Check spot exchange is enabled to make an order on spot market and if not revert
- Check order's price and quantity tick sizes fits market's min quantity and price tick size
- Increment subaccount's `TradeNonce`
- Reject if spot market id does not reference an active spot market
- Calculate unique order hash with `TradeNonce`
- Reject if the subaccount's available deposits does not have at least the required funds for the trade
- Decrement the available balance by the funds amount needed to fund the order
- Store the order in the transient limit order store and transient market indicator store

**Note:** The order in transient store is executed on endblocker or if not, put on long-live store.

## Batch creation of spot limit orders

Batch creation of spot limit orders is carried out by `MsgBatchCreateSpotLimitOrders` which consists of `Sender` and `Orders`.

**Steps**

- Loop over the `msg.Orders` and create spot limit order as in `MsgCreateSpotLimitOrder`

## Spot market order creation

Spot market order creation is carried out by `MsgCreateSpotMarketOrder` which consists of `Sender` and `Order`.

**Steps**

- Check spot exchange is enabled to make an order on spot market and if not revert
- Check order's price and quantity tick sizes fits market's min quantity and price tick size
- Increment subaccount's `TradeNonce`
- Reject if spot market id does not reference an active spot market
- Calculate unique order hash with `TradeNonce`
- Check available balance to fund the market order
- Calculate the worst acceptable price for the market order
- Decrement deposit's AvailableBalance by the balance hold
- Store the order in the transient spot market order store and transient market indicator store

## Cancel spot order

Spot order cancellation is carried out by `MsgCancelSpotOrder` which consists of `Sender` and `MarketId`, `SubaccountId` and `OrderHash`.

**Steps**

- Check spot exchange is enabled to execute the action and if not revert
- Reject if spot market id does not reference an active, suspended or demolished spot market
- Check spot limit order exists by `marketID`, `subaccountID` and `orderHash`
- Add back the margin hold to available balance
- Increment the available balance margin hold
- Delete the order state from ordersStore and ordersIndexStore
- Emit `EventCancelSpotOrder` event with marketID and order info

## Batch cancellation of spot orders

Batch cancellation of spot orders is carried out by `MsgBatchCancelSpotOrders` which consists of `Sender` and `Data`.

**Steps**

- Loop over the `msg.Data` and cancel spot order as in `MsgCancelSpotOrder`

## Derivative limit order creation

Derivative limit order creation is carried out by `MsgCreateDerivativeLimitOrder` which consists of `Sender` and `Order`.

**Steps**

- Check derivative exchange is enabled to make an order on derivative market and if not revert
- Reject if market order is already placed on the market by `subaccountID` (**Note:** Can't the market order and limit order core exist?)
- Get derivative market and markPrice by `marketID`
- Get orderbook metadata (`SubaccountOrderbookMetadata`) the for specified `marketID` and `subaccountID`
- Ensure limit order is valid:
  - Market config (market id and tick sizes)
  - Subaccount trade nonce
  - Subaccount max order count
  - If reduce-only order:
    - Position with valid quantity and opposite direction exists
    - If order would result in other reduce-only orders becoming stale, reject it
  - If limit order:
    - Enough subaccount deposits for margin hold
    - If order is in opposite direction of existing position and results in other reduce-only orders becoming stale, cancel the stale reduce-only orders
- Store the order in the transient limit order store and transient market indicator store
- Update orderbook metadata for subaccount

## Batch creation of derivative limit orders

Batch creation of derivative limit orders is carried out by `MsgBatchCreateDerivativeLimitOrders` which consists of `Sender` and `Orders`.

**Steps**

- Loop over the `msg.Orders` and create derivative limit order as in `MsgCreateDerivativeLimitOrder`

## Derivative market order creation

Derivative market order creation is carried out by `MsgCreateDerivativeMarketOrder` which consists of `Sender` and `Order`.

**Steps**

- Check derivative exchange is enabled to make an order on derivative market and if not revert
- Check if `SubaccountID` that is going to make new order has limit derivative order or market order and reject. **Note:** Perpetual market can't place two market orders or both limit / market orders at the same time?
- Check order's price and quantity tick sizes fits market's min quantity and price tick size
- Increment Subaccount's `TradeNonce`
- Reject if derivative market id does not reference an active derivative market
- Calculate unique order hash with `TradeNonce`
- Check that the market order worst price reaches the best opposing resting orderbook price
- Check Order/Position Margin amount
- 1. If it's reduce only order
- A. Check if position for `subaccountID` on the market is not nil
- B. Check that the order can close the position
- C. Reject if position.quantity - AggregateReduceOnlyQuantity - order.quantity < 0
- D. Set MarginHold as zero for no margin hold for selling positions
- 2. If it's not reduce only order
- A. Check available balance to fund the market order
- B. Reject if the subaccount's available deposits does not have at least the required funds for the trade
- C. Decrement deposit's AvailableBalance by the balance hold
- For an opposing position, if AggregateVanillaQuantity > position.quantity - AggregateReduceOnlyQuantity - order.FillableQuantity, the new reduce-only order might invalidate some existing reduce-only orders or itself be invalid, and do operations for that.
- Store the order in the transient derivative market order store and transient market indicator store

## Cancel derivative order

Derivative order cancellation is carried out by `MsgCancelDerivativeOrder` which consists of `Sender`, `MarketId`, `SubaccountId` and `OrderHash`.

**Steps**

- Check derivative exchange is enabled to execute the operation and if not revert
- Reject if derivative market id does not reference an active derivative market
- Check resting derivative limit order exists by `marketID`, `subaccountID` and `orderHash`
- Add back the margin hold to available balance
- Skip cancelling limit orders if their type shouldn't be cancelled
- Delete the order state from ordersStore, ordersIndexStore and subaccountOrderStore
- Update orderbook metadata for subaccount
- Emit `EventCancelDerivativeOrder` event with marketID and order info

## Batch cancellation of derivative orders

Batch cancellation of derivative orders is carried out by `MsgBatchCancelDerivativeOrders` which consists of `Sender` and `Data`.

**Steps**

- Loop over the `msg.Data` and cancel spot order as in `MsgCancelDerivativeOrder`

## Batch order updates

Batch updating orders is carried out by `MsgBatchUpdateOrders` which consists of `Sender` and `Orders`.

**Steps**

- Cancel all orders in all market id specified by `SpotMarketIdsToCancelAll` and `DerivativeMarketIdsToCancelAll` for specified subaccount id
- Loop over the `msg.SpotOrdersToCancel` and cancel spot limit order as in `MsgCancelSpotOrder`. If the cancel fails, continue to next order. The success of cancellations is reflected in the `MsgBatchUpdateOrdersResponse` as `SpotCancelSuccess`.
- Loop over the `msg.DerivativeOrdersToCancel` and cancel derivative limit order as in `MsgCancelDerivativeOrder`. If the cancel fails, continue to next order. The success of cancellations is reflected in the `MsgBatchUpdateOrdersResponse` as `DerivativeCancelSuccess`.
- Loop over the `msg.SpotOrdersToCreate` and create spot limit order as in `MsgCreateSpotOrder`. If the creation fails, continue to next order. Successful creations are reflected in the `MsgBatchUpdateOrdersResponse` as `SpotOrderHashes`.
- Loop over the `msg.DerivativeOrdersToCreate` and create derivative limit order as in `MsgCreateDerivativeOrder`. If the creation fails, continue to next order. Successful creations are reflected in the `MsgBatchUpdateOrdersResponse` as `DerivativeOrderHashes`.

## Transfer between subaccounts

Transfer between subaccounts is executed by `MsgSubaccountTransfer` which consists of `Sender`, `SourceSubaccountId`, `DestinationSubaccountId` and `Amount`.

**Steps**

- Withdraw deposit from `msg.SourceSubaccountId` for `msg.Amount`, if fail revert transaction
- Increment deposit of `msg.DestinationSubaccountId` by `msg.Amount`
- Emit event for `EventSubaccountBalanceTransfer` with `SrcSubaccountId`, `DstSubaccountId` and `msg.Amount`

**Note:** With subaccount transfer, no need to transfer actual coins from bank module but changing the records are enough.

## Transfer to external account

Transfer to external account is executed by `MsgExternalTransfer` which consists of `Sender`, `SourceSubaccountId`, `DestinationSubaccountId` and `Amount`.

**Steps**

- Withdraw deposit from `msg.SourceSubaccountId` for `msg.Amount`, if fail revert transaction
- Increment deposit of `msg.DestinationSubaccountId` by `msg.Amount`
- Emit event for `EventSubaccountBalanceTransfer` with `SrcSubaccountId`, `DstSubaccountId` and `msg.Amount`

**Note:** With subaccount transfer, no need to transfer actual coins from bank module but changing the records are enough.

1. Event should be different for subaccount transfer and external transfer.
2. There's no difference in subaccount transfer and external transfer, still need to keep different messages?

## Liquidating a position

Liquidating a position is executed by `MsgLiquidatePosition` which consists of `Sender`, `SubaccountId`, `MarketId` and `Order`.

**Steps**

- Check derivative exchange is enabled to liquidate a position on derivative market and if not revert
- Reject if derivative market id does not reference an active derivative market
- Get derivative market and markPrice by `marketID`
- Get position for `marketID` and `subaccountID`
- Calculate `liquidationPrice` and `bankruptcyPrice` from the position info
- Determine vaporize or liquidate and if not all of them, revert
- Cancel all reduce-only limit orders created by the position holder in the given market
- Apply funding and update position
- Cancel all market orders created by the position holder in the given market
- Check and increment subaccount nonce, compute order hash
- Calculate `liquidationOrder` hash
- Set the liquidation order into the storage
- Execute liquidation by matching position and liquidation order
- Handle differently based on the payout is positive or negative (insurance fund is involved here in calculation)
  - Positive Payout:
    1. Send half of the payout to liquidator (incentive for running liquidator bots)
    2. Send the other half to the insurance fund (incentive for participating in insurance funds)
  - Negative Payout - Four levels of escalation to retrieve the funds:
    1. From trader's available balance
    2. From trader's locked balance by cancelling his vanilla limit orders
    3. From the insurance fund
    4. Not enough funds available. Pause the market and add markets to the storage to be settled in next block, see `BeginBlocker` specs.
- If market is a perpetual market, upgrade VWAP data based on liquidation price and quantity
- If there's remaining in liquidation order, return back remains by cancelling order

## Increasing position margin

Increasing position margin is executed by `MsgIncreasePositionMargin` which consists of `Sender`, `SourceSubaccountId`, `DestinationSubaccountId`, `MarketId` and `Amount`.

**Steps**

- Check derivative exchange is enabled to increase position margin on derivative market and if not revert
- Reject if derivative market id does not reference an active derivative market
- Get deposit of `sourceSubaccountID`
- If `deposit.AvailableBalance` is lower than `msg.Amount`, revert
- Get position by `marketID` and `destinationSubaccountID` and if not exist, revert
- Reduce deposit amount of `sourceSubaccountID` by `msg.Amount`
- Increase position margin by `msg.Amount` and update position in the store

## Exchange enable proposal

The enable of market type is done by `ExchangeEnableProposal` which consists of `Title`, `Description` and `ExchangeType`.

**Steps**

- `ValidateBasic` for proposal
- If `p.ExchangeType` is spot market, enable spot exchange
- If `p.ExchangeType` is derivative market, enable derivative market

## Spot market launch proposal

Launch of spot market is handled by `SpotMarketLaunchProposal` which consists of `Title`, `Description`, `Ticker`, `BaseDenom`, `QuoteDenom`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- `ValidateBasic` for proposal
- Validate `BaseDenom` and `QuoteDenom` are valid
- Validate if same market does not exist by `msg.BaseDenom` and `msg.QuoteDenom`
- Calculate RelayerFeeShareRate based on exchange module params. **Note:** for INJ currency, relayer share rate is set to 100%
- Save spot market with calculated `ticker`, `baseDenom`, `quoteDenom`, `exchangeParams.DefaultSpotMakerFeeRate`, `exchangeParams.DefaultSpotTakerFeeRate`, `relayerFeeShareRate`, `minPriceTickSize`, `minQuantityTickSize`, `marketID`, and `MarketStatus_Active`.

## Perpetual market launch proposal

Perpetual market launch is handled by `PerpetualMarketLaunchProposal` which consists of `Title`, `Description`, `Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleScaleFactor`, `OracleType`, `MakerFeeRate`, `TakerFeeRate`, `InitialMarginRatio`, `MaintenanceMarginRatio`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- `ValidateBasic` for proposal
- Validate `quoteDenom`.
- Calculate `marketID` from `ticker`, `quoteDenom`, `oracleBase`, `oracleQuote`, `oracleType`
- Validate active or inactive perpetual market for `marketID` does not exist
- Try getting derivative market price to check price oracle by `oracleBase`, `oracleQuote`, `oracleScaleFactor`, `oracleType`
- Validate insurance fund exist for `marketID`
- Calculate `defaultFundingInterval`, `nextFundingTimestamp`, `relayerFeeShareRate` from `exchange` module params
- Execute `SetDerivativeMarketWithInfo` to set market info into the storage with `market`, `marketInfo` and `funding` objects

## Expiry futures market launch proposal

Expiry futures market launch is handled by `ExpiryFuturesMarketLaunchProposal` which consists of `Title`, `Description`, `Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleScaleFactor`, `OracleType`, `Expiry`, `MakerFeeRate`, `TakerFeeRate`, `InitialMarginRatio`, `MaintenanceMarginRatio`, `MinPriceTickSize` and `MinQuantityTickSize` fields.

**Steps**

- `ValidateBasic` for proposal
- Validate `quoteDenom`
- Calculate `marketID` from `p.Ticker`, `p.QuoteDenom`, `p.OracleBase`, `p.OracleQuote`, `p.OracleType` and `p.Expiry`
- Validate active or inactive expiry futures market for `marketID` does not exist
- If expiry time passed `ctx.BlockTime()` already, revert
- Try getting derivative market price to check price oracle by `oracleBase`, `oracleQuote`, `oracleScaleFactor`, `oracleType`
- Validate insurance fund exist for `marketID`
- Calculate RelayerFeeShareRate based on exchange module params. **Note:** for INJ currency, relayer share rate is set to 100%
- Execute `SetDerivativeMarketWithInfo` to set market info into the storage with `market`, `marketInfo` objects **Note:** TwapStartTimestamp is set to `expiry - thirtyMinutesInSeconds`.

## Spot market param update proposal

The update of spot market param is handled by `SpotMarketParamUpdateProposal` which consists of `Title`, `Description`, `MarketId`, `MakerFeeRate`, `TakerFeeRate`, `RelayerFeeShareRate`, `MinPriceTickSize`, `MinQuantityTickSize` and `Status`.

**Steps**

- `ValidateBasic` for proposal
- Get spot market by `p.MarketId` and if not exist, revert
- Reset the params for `MakerFeeRate`, `TakerFeeRate`, `RelayerFeeShareRate`, `MinPriceTickSize`, `MinQuantityTickSize` and `Status` if not empty, if empty keep as it is.
- Validate `MakerFeeRate` is bigger than `TakerFeeRate`.

## Derivative market param update proposal

Derivative market param update is handled by `DerivativeMarketParamUpdateProposal` which consists of `Title`, `Description`, `MarketId`, `InitialMarginRatio`, `MaintenanceMarginRatio`, `MakerFeeRate`, `TakerFeeRate`, `RelayerFeeShareRate`, `MinPriceTickSize`, `MinQuantityTickSize` and `Status`.

**Steps**

- `ValidateBasic` for proposal
- Validate Derivative market exists by `p.MarketId` and if not exist, revert
- Reset the params for `InitialMarginRatio`, `MaintenanceMarginRatio`, `MakerFeeRate`, `TakerFeeRate`, `RelayerFeeShareRate`, `MinPriceTickSize`, `MinQuantityTickSize` and `Status` if not empty, if empty keep as it is.
- Validate `MakerFeeRate` is bigger than `TakerFeeRate`.
- Validate `InitialMarginRatio` is bigger than `MaintenanceMarginRatio`.
- Schedule Derivative market param update and update finalization on Endblocker - **Note:** this is due to the orders update for derivative market param update - should make sure nothing panics here.

## Trading Rewards Campaign Launch Proposal

**Steps**

- `ValidateBasic` for proposal
- No existing campaign may exist.
- Campaign start timestamps must be in the future.
- Campaign quote denoms must exist.
- All start timestamps must match the duration.
- Set Campaign Data (Reward Pools, Info, Market Qualifications and Market Point Multipliers)
- Emit `EventTradingRewardCampaignUpdate`

## Trading Rewards Campaign Update Proposal

**Steps**

- `ValidateBasic` for proposal
- All `StartTimestamp` inside `CampaignRewardPoolsUpdates` must equal an existing campaign.
- `CampaignDurationSeconds` cannot be modified, but must match the current campaign.
- `CampaignRewardPoolsUpdates` cannot modify the current campaign and may contain nil values to delete a reward pool.
- Campaign start timestamps from `CampaignRewardPoolsAdditions` must be in the future.
- Any campaign quote denoms must exist.
- Delete Current Campaign Data (Info, Market Qualifications and Market Point Multipliers)
- Set Campaign Data (Info, Market Qualifications and Market Point Multipliers)
- Set Reward Pool Updates
- Set Reward Pool Additions
- Emit `EventTradingRewardCampaignUpdate`

## Fee Discount Schedule Proposal

**Steps**

- `ValidateBasic` for proposal
- If Current Fee Discount Schedule exists, delete it along with Market Qualifications
- Defined quote denoms must exist.
- If a restart of the fee cycle is required (bucket count, bucket duration or quote denoms changed), delete all account fee buckets and restart cycle.
- Set the first fee paid bucket timestamp as the current block time
- Set New Fee Discount Schedule, delete it along with Market Qualifications
- Set New Market Qualifications
