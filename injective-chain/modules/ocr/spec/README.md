# Ocr

## Abstract

OCR module is to store chainlink's OCR(Off-Chain Report) info into the chain storage.

Feed configuration is managed by module admin and report move to on-chain by transmitters and observers.
Transmitters and observers are rewarded in LINK token on the chain configured by governance.

While storing feed information, module provide hooks where oracle module can use for the calculation of cumulative price for futures market.

## Contents

1. [Concepts](01_concepts.md)
2. [State](02_state.md)
3. [Messages](03_messages.md)
4. [Proposals](04_proposals.md)
5. [Begin-Block](05_begin_block.md)
6. [Hooks](06_hooks.md)
7. [Events](07_events.md)
8. [Parameters](08_params.md)
