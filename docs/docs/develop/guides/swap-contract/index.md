# Atomic Order Token swap contract

The [swap contract](https://github.com/InjectiveLabs/swap-contract) allows an instant swap between two different tokens. Under the hood, it uses atomic orders to place market orders in one or more spot markets.

## Getting started

Anyone can instantiate an instance of the swap contract. There is version of this contract uploaded on Injective mainnet already, and can be found [here](https://explorer.injective.network/code/67/).

Before instantiating the contract, as the contract owner, you have three questions to answer:

### 1. Which address should be the fee recipient?

Since orders placed by the swap contract are orders in the [Injective Exchange Module](../../../develop/modules/Injective/exchange/), this means each order can have a fee recipient which can receive 40% of the trading fee. Typically, Exchange dApps will set the fee recipient as their own addresses.

### 2. What tokens should this contract support?

Every token available in the contract must have a route defined. Route refers to which markets `token A` will go through in order to get `token B`. For example, if you would like to support swapping between ATOM and INJ, then you would have to set route by providing the contract the market IDs of ATOM/USDT and INJ/USDT, so that the it knows the route of swapping between ATOM and INJ would be ATOM ⇔ USDT ⇔ INJ.

At this moment, the contract can only support markets quoted in USDT.

### 3. How much buffer should be provided to this contract?

As the contract owner, you also have to provide funds to the contract which will be used when the swap happens. The buffer is used by the contract when it place orders. If the user wants to swap a big amount or swap in an illiquid market, then more buffer is required. An error will occur when the contract buffer cannot satisfy the user's input amount.

At this moment, the buffer should only be USDT.

## Messages

### Instantiate

Initializes the contract state with the contract version and configuration details. The config includes an administrator address and a fee recipient address.

```rust
pub fn instantiate(
    deps: DepsMut<InjectiveQueryWrapper>,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response<InjectiveMsgWrapper>, ContractError>
```

### Execute

Handles different types of transactions and admin functions:

- SwapMinOutput: Swap with the minimum output quantity.
- SwapExactOutput: Swap with an exact output quantity.
- SetRoute: Set a swap route.
- DeleteRoute: Delete a swap route.
- UpdateConfig: Update the contract configuration.
- WithdrawSupportFunds: Withdraw the support funds from the contract.

```rust
pub fn execute(
    deps: DepsMut<InjectiveQueryWrapper>,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<InjectiveMsgWrapper>, ContractError>
```

### Reply

Handles the replies from other contracts or transactions.

```rust
pub fn reply(
    deps: DepsMut<InjectiveQueryWrapper>,
    env: Env,
    msg: Reply,
) -> Result<Response<InjectiveMsgWrapper>, ContractError>
```

### Query

Handles various queries to the contract:

- GetRoute: Get a specific swap route.
- GetOutputQuantity: Get the output quantity for a given input quantity.
- GetInputQuantity: Get the input quantity for a given output quantity.
- GetAllRoutes: Get all available swap routes.

```rust
pub fn query(deps: Deps<InjectiveQueryWrapper>, env: Env, msg: QueryMsg) -> StdResult<Binary>
```

## Repo

The complete GitHub repository for the swap contract can be found [here](https://github.com/InjectiveLabs/swap-contract).