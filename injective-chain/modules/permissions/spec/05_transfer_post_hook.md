# Transfer Post-Hook Implementation

This document explains the transfer post-hook feature in the permissions module.
The feature lets a denom namespace point to an EVM contract that is called after a
successful bank transfer of that denom.

The post-hook is intentionally separate from the existing transfer restriction
hooks. Restriction hooks decide whether a transfer may proceed. The post-hook is
called only after the bank keeper has already moved balances.

## High-Level Flow

```text
                     namespace for transferred denom
                     contains optional evm_post_hook
                                  |
                                  v
+----------+     +----------+     +-------------------+     +------------------+
| MsgSend  | --> | x/bank   | --> | permissions       | --> | EVM hook         |
| / send   |     | transfer |     | PostSendHook      |     | postTransfer(...)|
+----------+     +----------+     +-------------------+     +------------------+
                      |                      |                         |
                      |                      |                         v
                      |                      |              optional contract logs
                      |                      |                         |
                      v                      v                         v
              balances already        error is debug-logged    EventPostHookLogs
              updated before          and does not fail         preserves topics
              hook execution          the bank transfer         and data
```

## SDK Bank Hook Layer

The repository uses an Injective Cosmos SDK fork tagged with
`v0.50.14-v1.19.0-transfer-posthook`. That fork adds a bank post-send hook API:

```go
type PostSendHook func(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coin)
```

The bank keeper stores a `PostSendHookWrapper` and exposes:

```go
AppendPostSendHook(postHook types.PostSendHook)
```

`SendCoins` and `InputOutputCoins` both apply send restrictions first, execute
the bank balance movement, and then invoke the post-send hook per coin.

```text
x/bank SendCoins(from, to, [coinA, coinB])

  validate spendable balance
  for each coin:
    run send restriction
    move that coin to final receiver
    remember final receiver

  for each coin:
    postSendHook.Apply(from, finalReceiver, coin)
```

Important details:

- The hook receives one `sdk.Coin` at a time, not an `sdk.Coins` basket.
- For multi-coin sends, the hook is called once per coin after all balance
  movements have succeeded.
- If a send restriction rewrites the receiver, the post-hook receives the final
  rewritten receiver.
- Module-to-account, account-to-module, and module-to-module sends that delegate
  to `SendCoins` also pass through this path.

## Permissions Module Wiring

The permissions module registers both its pre-send restriction and post-send
hook when the app module is created.

```text
permissions AppModule construction

  bankKeeper.AppendSendRestriction(k.SendRestrictionFn)
  bankKeeper.AppendPostSendHook(k.PostSendHook)

                 |
                 v
       x/bank owns the call site
       permissions owns denom-specific behavior
```

Relevant code:

- `injective-chain/modules/permissions/module/module.go`
- `injective-chain/modules/permissions/types/expected_keepers.go`
- SDK fork: `x/bank/types/send_hooks.go`
- SDK fork: `x/bank/keeper/send.go`

## Namespace Configuration

The post-hook address is stored on the permissions namespace:

```proto
message Namespace {
  string denom = 1;
  string wasm_hook = 2;
  ...
  string evm_hook = 8;
  string evm_post_hook = 9;
}
```

`MsgUpdateNamespace` can set or clear it with the `evm_post_hook` field:

```proto
message MsgUpdateNamespace {
  message SetContractHook { string new_value = 1; }
  ...
  SetContractHook evm_hook = 8;
  SetContractHook evm_post_hook = 9;
}
```

`ValidateBasic` checks that non-empty EVM hook addresses are valid hex
addresses. The keeper then validates that the contract implements the expected
post-hook ABI before storing the namespace.

```text
CreateNamespace / UpdateNamespace

  ValidateBasic
    evm_post_hook must be empty or 0x hex address

  keeper validation
    call validateEvmPostHook(...)
      use cached context
      call postTransfer(dummyFrom, dummyTo, 1inj)
      discard validation state changes

  set namespace.evm_post_hook
```

Clearing the hook is done by setting `evm_post_hook.new_value` to an empty
string.

## Contract Interface

The EVM post-hook contract must implement:

```solidity
struct Coin {
    uint256 amount;
    string denom;
}

function postTransfer(
    address from,
    address to,
    Coin calldata amount
) external;
```

The Go binding used by the keeper is generated in
`injective-chain/modules/permissions/types/PermissionsPostHook.abigen.go`.

Address and amount conversion is direct:

```text
sdk.AccAddress from  -> common.BytesToAddress(from.Bytes())
sdk.AccAddress to    -> common.BytesToAddress(to.Bytes())
sdk.Coin amount      -> { amount: amount.Amount.BigInt(), denom: amount.Denom }
```

## Runtime Execution

The runtime entry point is `Keeper.PostSendHook` in
`injective-chain/modules/permissions/keeper/post_send.go`.

```text
PostSendHook(ctx, from, to, coin)

  namespace = GetNamespace(ctx, coin.Denom, false)

  if namespace missing:
    return

  if namespace.evm_post_hook empty:
    return

  callEvmPostHook(
    contract = namespace.evm_post_hook,
    from     = from as EVM address,
    to       = to as EVM address,
    amount   = coin.Amount,
    denom    = coin.Denom,
  )

  if error:
    log debug message
    return
```

`callEvmPostHook` then ABI-encodes `postTransfer`, builds an internal
`MsgEthereumTx`, and executes it with the EVM keeper:

```text
callEvmPostHook

  cap gas
  pack ABI call data for postTransfer(from, to, Coin)
  msg = evmtypes.NewTx(..., to = hookContract, gas = gasCap, input = callData)
  resp = evmKeeper.ApplyTransaction(execCtx, msg)

  if EVM execution failed:
    return hook error

  if resp.Logs not empty:
    emit EventPostHookLogs{
      post_hook_address,
      logs: [{ topics, data }]
    }
```

The bank hook type has no error return value, so post-hook execution errors do
not abort the already-successful transfer. They are logged at debug level by the
permissions keeper.

## Gas Handling

The post-hook shares the same gas parameter as the EVM and Wasm contract hooks:

```proto
uint64 contract_hook_max_gas = 1;
```

Default value:

```go
ContractHookMaxGas: 15_000_000
```

Gas handling in `callEvmPostHook` is:

```text
parent gas remaining == 0
  -> return hook error

parent gas meter is infinite
  -> execute hook with a fresh finite meter capped by ContractHookMaxGas

ContractHookMaxGas == 0
  -> return hook error; hook execution disabled

otherwise
  -> gasCap = min(parentGasRemaining, ContractHookMaxGas)
```

```text
                  +-----------------------+
                  | parent ctx gas meter  |
                  +-----------------------+
                              |
               +--------------+--------------+
               |                             |
               v                             v
        finite remaining               infinite remaining
               |                             |
               v                             v
 min(remaining, ContractHookMaxGas)   new finite gas meter
                                             |
                                             v
                                  ContractHookMaxGas limit
```

This prevents a post-hook from using unbounded gas when it is called from
consensus or other paths that may have an infinite SDK gas meter.

## Events And Logs

The EVM call can emit logs. The permissions keeper copies those logs into a
typed Cosmos event:

```proto
message EventPostHookLogs {
  string post_hook_address = 1;
  repeated EventPostHookLog logs = 2;
}

message EventPostHookLog {
  repeated string topics = 1;
  bytes data = 2;
}
```

This keeps hook contract logs visible from transaction and block results even
though the hook call is an internal EVM execution triggered by a bank transfer.

```text
EVM receipt logs
      |
      v
+-------------------+
| EventPostHookLogs |
+-------------------+
| post_hook_address |
| logs[]            |
|  - topics[]       |
|  - data           |
+-------------------+
      |
      v
CometBFT tx events / block results
```

## Failure Semantics

Post-hook failures are non-blocking at transfer time:

```text
bank transfer succeeds
        |
        v
post-hook runs
        |
   +----+----+
   |         |
   v         v
 success   error / revert / out of gas / panic
   |         |
   v         v
 emit logs  debug log only
```

Validation failures are blocking when configuring a namespace. A create or
update message that points `evm_post_hook` at a contract without a compatible
`postTransfer` implementation is rejected before the namespace is stored.

## Tests

The main coverage is:

- `injective-chain/modules/permissions/keeper/evm_hook_test.go`
  - Deploys a side-effect hook contract.
  - Creates a tokenfactory denom and namespace with `EvmPostHook`.
  - Sends the denom and verifies the hook contract counter increments.
  - Verifies that updating to an incompatible post-hook contract fails.
  - Verifies clearing the hook stops post-hook execution.
- `interchaintest/chain_permissions_post_hook_test.go`
  - Runs against a real chain.
  - Deploys an EVM post-hook contract.
  - Creates a namespace with `EvmPostHook`.
  - Sends the permissioned denom.
  - Verifies the contract side effect and the presence of `EventPostHookLogs`
    in block results.
