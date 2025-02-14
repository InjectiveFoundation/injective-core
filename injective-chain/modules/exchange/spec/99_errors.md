# Error Codes

This document lists the error codes used in the module.


| Module | Error Code | description |
|--------|------------|-------------|
| exchange |  1 | failed to validate order |
| exchange |  2 | spot market not found |
| exchange |  3 | spot market exists |
| exchange |  4 | struct field error |
| exchange |  5 | failed to validate market |
| exchange |  6 | subaccount has insufficient deposits |
| exchange |  7 | unrecognized order type |
| exchange |  8 | position quantity insufficient for order |
| exchange |  9 | order hash is not valid |
| exchange |  10 | subaccount id is not valid |
| exchange |  11 | invalid ticker |
| exchange |  12 | invalid base denom |
| exchange |  13 | invalid quote denom |
| exchange |  14 | invalid oracle |
| exchange |  15 | invalid expiry |
| exchange |  16 | invalid price |
| exchange |  17 | invalid quantity |
| exchange |  18 | unsupported oracle type |
| exchange |  19 | order doesnt exist |
| exchange |  20 | spot limit orderbook fill invalid |
| exchange |  21 | perpetual market exists |
| exchange |  22 | expiry futures market exists |
| exchange |  23 | expiry futures market expired |
| exchange |  24 | no liquidity on the orderbook! |
| exchange |  25 | Orderbook liquidity cannot satisfy current worst price |
| exchange |  26 | insufficient margin |
| exchange |  27 | Derivative market not found |
| exchange |  28 | Position not found |
| exchange |  29 | Position direction does not oppose the reduce-only order |
| exchange |  30 | Price Surpasses Bankruptcy Price |
| exchange |  31 | Position not liquidable |
| exchange |  32 | invalid trigger price |
| exchange |  33 | invalid oracle type |
| exchange |  34 | invalid minimum price tick size |
| exchange |  35 | invalid minimum quantity tick size |
| exchange |  36 | invalid minimum order margin |
| exchange |  37 | Exceeds order side count |
| exchange |  38 | Subaccount cannot place a market order when a market order in the same market was already placed in same block |
| exchange |  39 | cannot place a conditional market order when a conditional market order in same relative direction already exists |
| exchange |  40 | An equivalent market launch proposal already exists. |
| exchange |  41 | Invalid Market Status |
| exchange |  42 | base denom cannot be same with quote denom |
| exchange |  43 | oracle base cannot be same with oracle quote |
| exchange |  44 | MakerFeeRate does not match TakerFeeRate requirements |
| exchange |  45 | MaintenanceMarginRatio cannot be greater than InitialMarginRatio |
| exchange |  46 | OracleScaleFactor cannot be greater than MaxOracleScaleFactor |
| exchange |  47 | Spot exchange is not enabled yet |
| exchange |  48 | Derivatives exchange is not enabled yet |
| exchange |  49 | Oracle price delta exceeds threshold |
| exchange |  50 | Invalid hourly interest rate |
| exchange |  51 | Invalid hourly funding rate cap |
| exchange |  52 | Only perpetual markets can update funding parameters |
| exchange |  53 | Invalid trading reward campaign |
| exchange |  54 | Invalid fee discount schedule |
| exchange |  55 | invalid liquidation order |
| exchange |  56 | Unknown error happened for campaign distributions |
| exchange |  57 | Invalid trading reward points update |
| exchange |  58 | Invalid batch msg update |
| exchange |  59 | Post-only order exceeds top of book price |
| exchange |  60 | Order type not supported for given message |
| exchange |  61 | Sender must match dmm account |
| exchange |  62 | already opted out of rewards |
| exchange |  63 | Invalid margin ratio |
| exchange |  64 | Provided funds are below minimum |
| exchange |  65 | Position is below initial margin requirement |
| exchange |  66 | Pool has non-positive total lp token supply |
| exchange |  67 | Passed lp token burn amount is greater than total lp token supply |
| exchange |  68 | unsupported action |
| exchange |  69 | position quantity cannot be negative |
| exchange |  70 | binary options market exists |
| exchange |  71 | binary options market not found |
| exchange |  72 | invalid settlement |
| exchange |  73 | account doesnt exist |
| exchange |  74 | sender should be a market admin |
| exchange |  75 | market is already scheduled to settle |
| exchange |  76 | market not found |
| exchange |  77 | denom decimal cannot be higher than max scale factor |
| exchange |  78 | state is invalid |
| exchange |  79 | transient orders up to cancellation not supported |
| exchange |  80 | invalid trade |
| exchange |  81 | no margin locked in subaccount |
| exchange |  82 | Invalid access level to perform action |
| exchange |  83 | Invalid address |
| exchange |  84 | Invalid argument |
| exchange |  85 | Invalid funds direction |
| exchange |  86 | No funds provided |
| exchange |  87 | Invalid signature |
| exchange |  88 | no funds to unlock |
| exchange |  89 | No msgs provided |
| exchange |  90 | No msg provided |
| exchange |  91 | Invalid amount |
| exchange |  92 | The current feature has been disabled |
| exchange |  93 | Order has too much margin |
| exchange |  94 | Subaccount nonce is invalid |
| exchange |  95 | insufficient funds |
| exchange |  96 | exchange is in post-only mode |
| exchange |  97 | client order id already exists |
| exchange |  98 | client order id is invalid. Max length is 36 chars |
| exchange |  99 | market cannot be settled in emergency mode |
| exchange |  100 | invalid notional |
| exchange |  101 | stale oracle price |
| exchange |  102 | invalid stake grant |
| exchange |  103 | insufficient stake for grant |
| exchange |  104 | invalid permissions |
| exchange |  105 | the decimals specified for the denom is incorrect |
