---
sidebar_position: 2
title: State Transitions
---

# State Transitions

This document describes the state transition operations pertaining to:

- Creating an insurance fund
- Underwriting an insurance fund
- Request a redemption from the insurance fund
- Automatic processing of matured redemption requests

## Creating insurance fund

**Params description**
`Sender` field describes the creator of an insurance fund .
`Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleType`, `Expiry` fields describe the derivative market info
that the insurance fund associated to.
`InitialDeposit` field describes the initial deposit amount to be put on the insurance fund.

**Steps**

- Get `MarketId` for the insurance fund - **Note**, market could be not available yet on `exchange` and it's not an
  issue
- Ensure if insurance fund associated to the `MarketId` does not exist
- Ensure if initial deposit amount is not zero
- Get `shareDenom` that is unique - it's incremented when share denom is requested for insurance fund creation or when
  underwriting insurance fund that has zero balance and non-zero total share denom supply.
- Send coins from creator's account to insurance fund module account
- Create insurance fund object with `DefaultRedemptionNoticePeriodDuration` and with the params provided
- Set `Balance` of fund object to initial deposit amount
- Mint `InsuranceFundInitialSupply` (10^18) `shareDenom` tokens to creator account
- Save insurance fund object to store
- Register newly created insurance fund `shareDenom` metadata inside BankKeeper

## Underwriting an insurance fund

**Params description**
`Sender` field describes the underwriter of an insurance fund .
`MarketId` field describes the derivative market id to the insurance fund.
`Deposit` field describes the deposit amount to be added on the insurance fund.

**Steps**

- Ensure if insurance fund associated to the `MarketId` does exist
- Send underwriting tokens from sender's account to module account
- Make actions based on the status of insurance fund associated to the `MarketId`.
    * A. when `Balance` and `ShareDenomSupply` are zero
        1. mint `InsuranceFundInitialSupply` (10^18) to the sender.
        2. set `Balance` to deposit amount
        3. set `ShareDenomSupply` to `InsuranceFundInitialSupply`
    * B. when `Balance` is zero and `ShareDenomSupply` is not zero
        1. change `ShareDenom` of the the insurance fund to start new insurance fund from beginning.
        2. register newly created `ShareDenom` in bank keeper
        3. mint `InsuranceFundInitialSupply` (10^18) to the sender.
        4. set `Balance` to deposit amount
        5. set `ShareDenomSupply` to `InsuranceFundInitialSupply`
    * C. when `Balance` is not zero and `ShareDenomSupply` is zero
        1. mint `InsuranceFundInitialSupply` (10^18) to the sender.
        2. increase `Balance` by deposit amount
        3. set `ShareDenomSupply` to `InsuranceFundInitialSupply`
    * D. when both `Balance` and `ShareDenomSupply` are not zero - normal case
        1. increase `Balance` by deposit amount
        2. mint `prev_ShareDenomSupply * deposit_amount / prev_Balance` amount of `ShareDenom` to sender
        3. increase `ShareDenomSupply` with mint amount
- Save insurance fund object to store

## Requesting a redemption from an insurance fund

**Params description**
`Sender` field describes the redemption requester of an insurance fund .
`MarketId` field describes the derivative market id associated to the insurance fund.
`Amount` field describes the share token amount to be redeemed.

**Steps**

- Ensure insurance fund associated to the `MarketId` does exist
- Send `ShareDenom` to module account
- Get new redemption schedule ID
- Calculate `ClaimTime` from insurance fund's redemption notice period duration and current block time
- Calculate key to store pending redemption (redemption schedule)
- Create redemption schedule object with details
- Store redemption schedule object to store

## Insurance fund actions on liquidation events in derivative market

**Steps**

- `exchange` module finds relative insurance fund from the insurance keeper.
- if `missingFund` is positive, it withdraws the amount from the insurance fund through `WithdrawFromInsuranceFund`.
- if `missingFund` is negative, it deposits the amount into the insurance fund through `DepositIntoInsuranceFund`.

## Automatic processing of pending redemptions

**Steps**

Iterate all matured redemptions by sorted order by `ClaimTime` and perform the following actions:

- If `ClaimTime` is after current block time, break early
- Ensure the insurance fund exist for matured redemption schedule
- Calculate redeem amount from share amount - `shareAmt * fund.Balance * fund.TotalShare`
- Send calculate redeem amount from module account to redeemer account
- Burn share tokens sent to the module account at the time of redemption schedule
- Delete redemption schedule object
- Reduce insurance fund's `Balance` by redeem amount
- Store updated insurance object to store

# Hooks

Other modules may register operations to execute when a certain event has occurred within insurance fund. These events
can be registered to execute either right `Before` or `After` the exchange event (as per the hook name). The following
hooks can registered with the exchange:

**Note**: Hooks are not available and exchange module calls insurance keeper function directly.

**Steps**
When liquidation event happen in derivative market

- `exchange` module finds relative insurance fund from the insurance keeper.
- if `missingFund` is positive, it withdraws the amount from the insurance fund through `WithdrawFromInsuranceFund`.
- if `missingFund` is negative, it deposits the amount into the insurance fund through `DepositIntoInsuranceFund`.

