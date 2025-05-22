# Txfees

The txfees module for Injective provides the required functionality to support fee market as per EIP-1559.

EIP-1559 introduces a "base fee" that automatically adjusts based on network congestion. When network activity increases, the base fee increases, and when activity decreases, the base fee decreases. This creates a more predictable and efficient fee market compared to a simple first-price auction model.

For more details, see the official EIP-1559 specification: https://eips.ethereum.org/EIPS/eip-1559

## Module Parameters

The txfees module's parameters control both transaction acceptance rules and the EIP-1559 fee market behavior. These parameters can be updated through governance.

### Transaction Control Parameters

These parameters define the basic transaction validation rules that are always enforced, regardless of whether the EIP-1559 fee market is enabled (`Mempool1559Enabled`). They provide the first line of defense against network spam by setting hard limits on transaction characteristics and implementing a two-tiered fee system for high-gas transactions.

When `Mempool1559Enabled` is false, these are the only parameters used for transaction validation. When true, these checks are performed before the EIP-1559 fee market rules are applied.

#### MaxGasWantedPerTx
- Type: `uint64`
- Default: `30,000,000`
- Description: Maximum gas allowed per transaction. Transactions with gas limit higher than this will be rejected from the mempool.

#### HighGasTxThreshold
- Type: `uint64`
- Default: `2,500,000`
- Description: Gas threshold above which a transaction is considered "high gas". When a transaction's gas exceeds this threshold, it must pay at least `MinGasPriceForHighGasTx` as the gas price.

#### MinGasPriceForHighGasTx
- Type: `sdk.Dec`
- Default: `0`
- Description: Minimum gas price required for high gas transactions. Transactions above the `HighGasTxThreshold` must have at least this gas price to be accepted into the mempool.

### Fee Market Parameters

These parameters control the dynamic EIP-1559 fee market behavior and are only active when `Mempool1559Enabled` is true. They determine how the base fee adjusts in response to network congestion, set the bounds for fee adjustments, and define the target block utilization.

When `Mempool1559Enabled` is false, these parameters (except `MinGasPrice`) are not used, and transactions only need to meet the basic transaction control requirements. When true, transactions must additionally satisfy the EIP-1559 fee market rules, including providing a gas price ≥ current base fee.

Note: `MinGasPrice` is always enforced as the minimum gas price, regardless of whether EIP-1559 is enabled.

#### Mempool1559Enabled
- Type: `bool`
- Default: `false`
- Description: Enables EIP-1559 style adaptive fee market in the mempool. When enabled, the base fee automatically adjusts based on network congestion.

#### MinGasPrice
- Type: `sdk.Dec`
- Default: `160,000,000` (INJ)
- Description: The minimum allowed base fee. The base fee cannot drop below this value, providing a price floor for transaction fees.

#### DefaultBaseFeeMultiplier
- Type: `sdk.Dec`
- Default: `1.5`
- Description: Multiplier applied to `MinGasPrice` to calculate the default base fee. The default base fee (`MinGasPrice` * `DefaultBaseFeeMultiplier`) is used when resetting the fee market every reset interval.

#### MaxBaseFeeMultiplier
- Type: `sdk.Dec`
- Default: `1000`
- Description: Maximum multiplier applied to `MinGasPrice` to calculate the maximum allowed base fee. This prevents fees from becoming excessively high.

#### ResetInterval
- Type: `int64`
- Default: `36,000` (blocks, approximately 8 hours)
- Description: The interval at which the base fee is reset to the default base fee. This prevents long-term fee drift and ensures periodic resets to a known baseline.

#### MaxBlockChangeRate
- Type: `sdk.Dec`
- Default: `0.1` (10%)
- Description: The maximum rate at which the base fee can change per block. This limits fee volatility between blocks.
- Calculation: The base fee adjustment uses the following formula:
  ```
  baseFeeMultiplier = 1 + (gasUsed - targetGas) / targetGas * maxChangeRate
  newBaseFee = currentBaseFee * baseFeeMultiplier
  ```
  Where:
  - `gasUsed` is the total gas consumed in the block
  - `targetGas` is determined by `TargetBlockSpacePercentRate` * block gas limit
  - `maxChangeRate` is 0.1 (10%)
- Impact: 
  - When block is full (gasUsed = block gas limit): Base fee increases by ~6%
  - When block is empty (gasUsed = 0): Base fee decreases by 10%
  - When gasUsed = targetGas: No change in base fee
  - The asymmetric change rates (6% up vs 10% down) help fees recover more quickly after congestion
- Transaction Processing:
  - In CheckTx: New transactions must provide a gas price ≥ current base fee to be accepted
  - In RecheckTx: 
    - For low base fees (≤ 4x MinGasPrice): Transactions with gas price < current base fee / 2 are removed
    - For high base fees (> 4x MinGasPrice): Transactions with gas price < current base fee / 2.3 are removed
  - This dual-threshold approach helps maintain network stability during normal operation while allowing faster recovery during congestion

#### TargetBlockSpacePercentRate
- Type: `sdk.Dec`
- Default: `0.625` (62.5%)
- Description: Target percentage of the block gas limit that should be used. When actual usage exceeds this target, the base fee increases. When usage is below target, the base fee decreases.

### Fee Recheck Parameters

These parameters control the mempool transaction eviction mechanism and are only relevant when `Mempool1559Enabled` is true. They determine when existing transactions should be removed from the mempool as the base fee changes, implementing a dual-threshold approach that balances network stability with congestion recovery.

When `Mempool1559Enabled` is false, transactions in the mempool are not rechecked against changing base fees. When true, these parameters work in conjunction with the Fee Market Parameters to maintain mempool health by ensuring transactions remain economically viable as network conditions change.

The recheck mechanism uses different thresholds for low and high base fee scenarios:
- In low base fee conditions: Focuses on network stability with more conservative eviction rules
- In high base fee conditions: Prioritizes quick recovery from congestion with more aggressive eviction

#### RecheckFeeLowBaseFee
- Type: `sdk.Dec`
- Default: `3.0`
- Description: When the base fee is low (≤ 4x MinGasPrice), transactions must have fees at least 1/3 of the current base fee to remain in the mempool. This more conservative multiplier at lower fee levels helps maintain network stability by preventing too rapid eviction of transactions. If spam transactions are detected, it takes approximately 19 blocks from when the base fee exceeds the spam cost until those transactions are evicted from the mempool.

#### RecheckFeeHighBaseFee
- Type: `sdk.Dec`
- Default: `2.3`
- Description: When the base fee is high (> 4x MinGasPrice), transactions must have fees at least 1/2.3 of the current base fee to remain in the mempool.

#### RecheckFeeBaseFeeThresholdMultiplier
- Type: `sdk.Dec`
- Default: `4.0`
- Description: Multiplier applied to `MinGasPrice` to determine the threshold between high and low base fee regimes for recheck purposes. The threshold is `MinGasPrice` * `RecheckFeeBaseFeeThresholdMultiplier`.

## Modifying Module Parameters

The txfees module parameters can be modified through governance proposals. This ensures that any changes to these critical parameters are approved by the community. Here's how to modify these parameters:

### Through Governance Proposal

Parameters can be updated using a `MsgUpdateParams` transaction wrapped in a governance proposal. The proposal must be submitted by the governance module account.

Example of updating multiple parameters:
```json
{
  "title": "Update Txfees Parameters",
  "description": "Adjust fee market parameters to improve network performance during congestion",
  "messages": [
    {
      "@type": "/injective.txfees.v1beta1.MsgUpdateParams",
      "authority": "inj10d07y265gmmuvt4z0w9aw880jnsr700jvss730",  // gov module account
      "params": {
        "max_gas_wanted_per_tx": "100000000",             // Increase max gas per tx to 100M
        "min_gas_price": "200000000",                     // Increase min gas price to 200M INJ
        "default_base_fee_multiplier": "2.0",             // Increase default multiplier to 2.0
        "max_block_change_rate": "0.15",                  // Increase max change rate to 15%
        "target_block_space_percent_rate": "0.75",        // Increase target utilization to 75%
        "recheck_fee_low_base_fee": "3.5",               // Increase low base fee recheck threshold
        // ... other parameters remain unchanged ...
      }
    }
  ],
  "deposit": "1000000000000000000inj"  // Example deposit
}
```

### Parameter Validation

When updating parameters:
1. All parameters must be provided in the update message (unchanged parameters should keep their current values)
2. Parameters are validated before being applied:
   - Numeric values must be positive
   - Multipliers and rates must be valid decimals
   - Thresholds must maintain logical relationships (e.g., `MinGasPrice` ≤ default base fee ≤ max base fee)

### Query Current Parameters

You can query current parameter values using the gRPC endpoint:
```bash
/injective/txfees/v1beta1/params
```

You can also query the current EIP-1559 base fee using the CLI:
```bash
injectived query txfees base-fee
```

### Query Current Base Fee

The current EIP-1559 base fee can be queried through multiple interfaces:

#### CLI
```bash
injectived query txfees base-fee
```

#### gRPC
The base fee can be queried using the `GetEipBaseFee` RPC method:

```protobuf
// Request
message QueryEipBaseFeeRequest {
}

// Response
message QueryEipBaseFeeResponse {
  EipBaseFee base_fee = 1;
}

message EipBaseFee {
  string base_fee = 1 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
}
```

Example using `grpcurl`:
```bash
grpcurl -plaintext localhost:9090 injective.txfees.v1beta1.Query/GetEipBaseFee

# Osmosis-like path for compatibility with IBC relayers and wallets
grpcurl -plaintext localhost:9090 osmosis.txfees.v1beta1.Query/GetEipBaseFee
```



Service Definition:
```protobuf
service Query {
  // Returns the current fee market EIP base fee
  rpc GetEipBaseFee(QueryEipBaseFeeRequest) returns (QueryEipBaseFeeResponse) {
    option (google.api.http).get = "/injective/txfees/v1beta1/cur_eip_base_fee";
  }
}
```

#### gRPC-Gateway (REST)
```bash
curl -X GET "http://localhost:1317/injective/txfees/v1beta1/cur_eip_base_fee"
```

#### Response Format
The response will contain the current base fee in INJ units. Example:
```json
{
  "base_fee": {
    "base_fee": "160000000"
  }
}
```

Note: The base fee is returned as a decimal string. When `Mempool1559Enabled` is false, this will return the `MinGasPrice` value.
