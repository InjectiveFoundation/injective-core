---
sidebar_position: 4
title: Events
---

# Events

The erc20 module emits the following events:

```protobuf 
message EventCreateTokenPair {
  string bank_denom = 1;
  string erc20_address = 2;
}

message EventDeleteTokenPair {
  string bank_denom = 1;
}
```