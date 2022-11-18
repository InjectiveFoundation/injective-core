---
sidebar_position: 3
title: State 
---

## State

### Last Code ID

- type: `uint64`

A counter for the last uploaded code ID.

### Last Instance ID

- type: `uint64`

A counter for the last instantiated contract number.

### Code

- type: `map[uint64]CodeInfo`

Maps a code ID to `CodeInfo` entry.

### Contract Info

- type: `map[bytes]ContractInfo`

Maps contract address to its corresponding `ContractInfo`.

### Contract Store

- type: `map[bytes]KVStore`

Maps contract address to its dedicated KVStore.