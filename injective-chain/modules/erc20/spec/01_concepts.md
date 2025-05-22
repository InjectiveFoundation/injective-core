---
sidebar_position: 1
title: Concepts
---

# Concepts

The `ERC20` module is designed to introduce existing bank denoms (such as IBC-bridged tokens, USDC, tokenfactory, and peggy) into the Injective EVM. It achieves this by maintaining an association between token pairs in its storage. When creating a new token pair for an existing bank denom, the module uploads a fresh ERC20 contract that utilizes our bank precompile. This precompile then accesses the module's storage to locate the mapping between the newly created ERC20 address and the existing bank denom. We require this separate module for several reasons:

1. storage: store mapping between bank denom ↔ erc20 address
2. new Msg type: allow users to create new token pairs in the mappings, which is done by issuing a chain Msg
