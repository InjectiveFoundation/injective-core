---
sidebar_position: 3
---

# Messages

In this section we describe the processing of the `erc20` module messages and the corresponding updates to the state.

### Create Token Pair: `MsgCreateTokenPair`

Creates an association between existing bank denom and new or existing ERC20 smart contract.
If ERC20 address is empty, new ERC20 smart contract will be instantiated. Not all bank denoms are supported.

Validation rules:

- for tokenfactory denoms, only denom admin can create token pair
- for peggy and IBC denoms, anyone can create token pair (only with erc20 address being empty)

```go
type MsgCreateTokenPair struct {
	Sender    string   
	TokenPair TokenPair
}

type TokenPair struct {
	BankDenom    string
	Erc20Address string
}
```

**State Modifications:**

- Validation checks:
	- Sender has permissions to create token pair for this denom (for tokenfactory denom it must be a denom admin)
	- Provided bank denom exists and has non-zero supply
	- If ERC20 address is provided:
		- check that contract exists and is, in fact, an ERC-2o smart contract (by invoking `symbol()` method on it)
		- check that existing contract does not have associated bank denom already with circulating supply
- Create the association depending on the bank denom type:
	- tokenfactory denom:
		- if no ERC-20 address is provided, instantiate new `MintBurnBankERC20` smart contract, otherwise use provided address
		- store the association
	- IBC and peggy denoms:
		- instantiate new `FixedSupplyBankERC20` smart contract
		- store the association

### Delete Token Pair: `MsgDeleteTokenPair`

Only authority can remove token pairs for now, by providing bank denom of the pair.

```go
type MsgDeleteTokenPair struct {
	Sender    string
	BankDenom string
}
```
