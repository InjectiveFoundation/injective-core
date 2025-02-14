# `Insurance`

## Abstract

This paper specifies the insurance module of the Injective Chain. 

This module provides insurance funds for derivative markets in the `exchange` module of the Injective Chain to use in order to support higher leverage trading. On a high level, insurance funds for each derivative market are funded by a permissionless group of underwriters who each own a proportional claim (represented through insurance fund share tokens) over the underlying assets in the insurance fund.

Each insurance fund grows when positions in its corresponding derivative market are liquidated with positive equity, as half of the positive equity is sent to the insurance fund upon liquidation. When a position with negative equity is liquidated (i.e. the position has surpassed bankruptcy), the insurance fund is utilized to cover the missing equity. 

## Contents

1. [State](01_state.md)
2. [State Transitions](02_state_transitions.md)
3. [Messages](03_messages.md)
4. [End Block](04_end_block.md)
5. [Events](05_events.md)
6. [Params](06_params.md)
7. [Future Improvements](07_future_improvements.md)
