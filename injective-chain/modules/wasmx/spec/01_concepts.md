---
sidebar_position: 1
title: Concepts 
---

## Concepts

### Smart contracts

Smart contracts are autonomous agents that can interact with other entities on the Injective blockchain, such as human-owned accounts, validators, and other smart contracts. Each smart contract has:

- A unique **contract address** with an account that holds funds.
- A **code ID**, where its logic is defined.
- Its own **key-value store**, where it can persist and retrieve data.

#### Contract address

Upon instantiation, each contract is automatically assigned an Injective account address, called the contract address. The address is procedurally generated on-chain without an accompanying private and public key pair, and it can be completely determined by the contract's number order of existence. For instance, on two separate Injective networks, the first contract will always be assigned the address `inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y`, and similarly for the second, third, and so on.

#### Code ID

On Injective, code upload and contract creation are separate events. A smart contract writer first uploads WASM bytecode onto the blockchain to obtain a code ID, which they then can use to initialize an instance of that contract. This scheme promotes efficient storage because most contracts share the same underlying logic and vary only in their initial configuration. Vetted, high-quality contracts for common use cases like fungible tokens and multisig wallets can be easily reused without the need to upload new code.

#### Key-value store

Each smart contract is given its own dedicated keyspace in LevelDB, prefixed by the contract address. Contract code is safely sandboxed and can only set and delete new keys and values within its assigned keyspace.

### Interaction

You can interact with smart contracts in several ways.

#### Instantiation

You can instantiate a new smart contract by sending a `MsgInstantiateContract`. In it, you can:

- Assign an owner to the contract.
- Specify code will be used for the contract via a code ID.
- Define the initial parameters / configuration through an `InitMsg`.
- Provide the new contract's account with some initial funds.
- Denote whether the contract is migratable (i.e. can change code IDs).

The `InitMsg` is a JSON message whose expected format is defined in the contract's code. Every contract contains a section that defines how to set up the initial state depending on the provided `InitMsg`.

#### Execution

You can execute a smart contract to invoke one of its defined functions by sending a `MsgExecuteContract`. In it, you can:

- Specify which function to call with a `HandleMsg`.
- Send funds to the contract, which may be expected during execution.

The `HandleMsg` is a JSON message that contains function call arguments and gets routed to the appropriate handling logic. From there, the contract executes the function's instructions during which the contract's own state can be modified. The contract can only modify outside state, such as state in other contracts or modules, after its own execution has ended, by returning a list of blockchain messages, such as `MsgSend` and `MsgSwap`. These messages are appended to the same transaction as the `MsgExecuteContract`, and, if any of the messages are invalid, the whole transaction is invalidated.

#### Migration

If a user is the contract's owner, and a contract is instantiated as migratable, they can issue a `MsgMigrateContract` to reset its code ID to a new one. The migration can be parameterized with a `MigrateMsg`, a JSON message.

#### Transfer of ownership

The current owner of the smart contract can reassign a new owner to the contract with `MsgUpdateContractOwner`.

#### Query

Contracts can define query functions, or read-only operations meant for data-retrieval. Doing so allows contracts to expose rich, custom data endpoints with JSON responses instead of raw bytes from the low-level key-value store. Because the blockchain state cannot be changed, the node can directly run the query without a transaction.

Users can specify which query function alongside any arguments with a JSON `QueryMsg`. Even though there is no gas fee, the query function's execution is capped by gas determined by metered execution, which is not charged, as a form of spam protection.

### Wasmer VM

The actual execution of WASM bytecode is performed by [wasmer](https://github.com/wasmerio/wasmer), which provides a lightweight, sandboxed runtime with metered execution to account for the resource cost of computation.

#### Gas meter

In addition to the regular gas fees incurred from creating the transaction, Injective also calculates a separate gas when executing smart contract code. This is tracked by the **gas meter**, which is during the execution of every opcode and gets translated back to native Injective gas via a constant multiplier (currently set to 100).

### Gas fees

WASM data and event spend gas up to `1 * bytes`. Passing the event and data to another contract also spends gas in reply.