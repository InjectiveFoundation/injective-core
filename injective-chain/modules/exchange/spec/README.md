# `Exchange`

## Abstract

The `exchange` module is the heart of the Injective Chain which enables fully decentralized spot and derivative exchange.  
It is the _sine qua non_ module of the chain and integrates tightly with the `auction`, `insurance`, `oracle`, and `peggy` modules.

The exchange protocol enables traders to create and trade on arbitrary spot and derivative markets.
The entire process of orderbook management, trade execution, order matching and settlement occurs on chain through the logic codified by the exchange module.

The `exchange` module enables the exchange of tokens on two types of markets:

1. `Derivative Market`: Either a `Perpetual Swap Market` or a `Futures Market`.
2. `Spot Market`

## Contents

1. **[Derivative Market Concepts](00_derivative_market_concepts.md)**
2. **[Spot Market Concepts](01_spot_market_concepts.md)**
3. **[Other Concepts](02_other_concepts.md)**
4. **[State](03_state.md)**
5. **[State Transitions](04_state_transitions.md)**
6. **[Messages](05_messages.md)**
7. **[Proposals](06_proposals.md)**
8. **[Begin Block](07_begin_block.md)**
9. **[End Block](08_end_block.md)**
10. **[Events](09_events.md)**
11. **[Params](10_params.md)**
12. **[MsgPrivilegedExecuteContract](11_msg_privileged_execute_contract.md)**
