<!--
order: 8
-->

# Parameters

The evm module contains the following parameters:

## Params

| Key            | Type        | Default Value   |
| -------------- | ----------- |-----------------|
| `EVMDenom`     | string      | `"inj"`         |
| `EnableCreate` | bool        | `true`          |
| `EnableCall`   | bool        | `true`          |
| `ExtraEIPs`    | []int       | TBD             |
| `ChainConfig`  | ChainConfig | See ChainConfig |

## EVM denom

The evm denomination parameter defines the token denomination used on the EVM state transitions and gas consumption for EVM messages.

For example, on Ethereum, the `evm_denom` would be `ETH`. To maintain parity with Ethereum, Injective uses Atto as its base denomination. In essence, 1 (atto) inj equals `1x10⁻¹⁸ INJ`, aligning with Ethereum’s denomination where one wei is equal to 1x10⁻¹⁸ ETH. In terms of precision, the `INJ` and `ETH` share the same value, *i.e* `1 INJ = 10^18 inj` and `1 ETH = 10^18 wei`.

## Enable Create

The enable create parameter toggles state transitions that use the `vm.Create` function. When the parameter is disabled, it will prevent all contract creation functionality.

## Enable Transfer

The enable transfer toggles state transitions that use the `vm.Call` function. When the parameter is disabled, it will prevent transfers between accounts and executing a smart contract call.

## Extra EIPs

The extra EIPs parameter defines the set of activateable Ethereum Improvement Proposals (**[EIPs](https://ethereum.org/en/eips/)**)
on the Ethereum VM `Config` that apply custom jump tables.

::: tip
NOTE: some of these EIPs are already enabled by the chain configuration, depending on the hard fork number.
:::

The supported activateable EIPS are:

- **[EIP 1344](https://eips.ethereum.org/EIPS/eip-1344)**
- **[EIP 1884](https://eips.ethereum.org/EIPS/eip-1884)**
- **[EIP 2200](https://eips.ethereum.org/EIPS/eip-2200)**
- **[EIP 2315](https://eips.ethereum.org/EIPS/eip-2315)**
- **[EIP 2929](https://eips.ethereum.org/EIPS/eip-2929)**
- **[EIP 3198](https://eips.ethereum.org/EIPS/eip-3198)**
- **[EIP 3529](https://eips.ethereum.org/EIPS/eip-3529)**

## Chain Config

The `ChainConfig` is a protobuf wrapper type that contains the same fields as the go-ethereum `ChainConfig` parameters, but using `*sdk.Int` types instead of `*big.Int`.

By default, all block configuration fields but `ConstantinopleBlock`, are enabled at genesis (height 0).

### ChainConfig Defaults

| Name                | Default Value                                                        |
| ------------------- | -------------------------------------------------------------------- |
| HomesteadBlock      | 0                                                                    |
| DAOForkBlock        | 0                                                                    |
| DAOForkSupport      | `true`                                                               |
| EIP150Block         | 0                                                                    |
| EIP150Hash          | `0x0000000000000000000000000000000000000000000000000000000000000000` |
| EIP155Block         | 0                                                                    |
| EIP158Block         | 0                                                                    |
| ByzantiumBlock      | 0                                                                    |
| ConstantinopleBlock | 0                                                                    |
| PetersburgBlock     | 0                                                                    |
| IstanbulBlock       | 0                                                                    |
| MuirGlacierBlock    | 0                                                                    |
| BerlinBlock         | 0                                                                    |
| LondonBlock         | 0                                                                    |
| ArrowGlacierBlock   | 0                                                                    |
| GrayGlacierBlock    | 0                                                                    |
| MergeNetsplitBlock  | 0                                                                    |
| ShanghaiTime        | 0                                                                    |
| CancunTime          | 0                                                                    |
| PragueTime          | 0                                                                    |
