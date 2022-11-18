---
sidebar_position: 1
title: Concepts
---

# Concepts

The `ocr` module is to store chainlink's OCR information into on-chain by verified members.

Off-chain reporting consists of N nodes (oracles), gathering data from external sources. Reports are being exchanged in a p2p fashion between oracles to get signatures of approval. A subset of nodes (transmitters) is identified by the `ocr` module on-chain, they must submit the reports to module, the first transmitter who hits the chain gets an extra reward to cover gas costs. Other transmitters are not. All oracles participating in the round are getting paid. `ocr` module stores median value from the reports.

## OCR Terminology

The protocol periodically sends **oracle reports** to the OCR module. The reporting protocol is comprised of three components: **pacemaker**, **report generation** and **transmission**. 

**Pacemaker**

The pacemaker drives the report generation process which is structured in **epochs**. Each epoch has a designatd leader who the pacemaker then tasks with starting the report generation protocol. If the leader does not produce a valid report in time, the pacemaker also aborts the current report generation and starts a new epoch. 

**Report Generation**

For a given epoch, the report generation protocol enters into **rounds** where **observations** are gathered and (given conditions are met such as heartbeat and deviation) a signed oracle **report** is generated. The rounds are controlled by a leader node who controls the frequency of rounds, gathers the observations and generates the report. 

**Transmission**

The transmission protocol then transmits the generated report to the OCR module.

## Off-chain OCR integration

- Provide means to communicate with Injective using sdk-go
- Read data from the module, such as a list of approved oracles
- Submit reports as Msgs (Implement `ContractTransmitter`)
- Implement `OffchainConfigDigester`
- Implement `OnchainKeyring` for producing signatures that will work on the target chain module
- Implement `ContractConfigTracker` for tracking changes of the chain module config (gov approved)

Notes:

- Reports are timestamped in Epoch-Round fashion
- `ocr` module verifies the signatures of oracles on the report
- `ocr` module records oracles who contributed to a report, for the payout
- `ocr` module stores the median of the observations
- `ocr` module provides extra reward for the first submitter of a Msg

### Integration Overview

Chainlink has several [price data feeds](https://data.chain.link/ethereum/mainnet/stablecoins) including:

- 80 Crypto/USD pairs (e.g. ETH/USD, BTC/USD)
- 17 Stablecoin pairs (e.g. USDT/USD, USDC/USD)
- 73 ETH pairs (e.g. LINK/ETH)
- 17 Forex pairs (e.g. GBP/USD, CNY/USD)

A derivative market on Injective specifies the following oracle parameters:

- An oracleBase (e.g. BTC)
- An oracleQuote (e.g. USDT)
- An oracleType (e.g. Chainlink)

Thus for a BTC/USDT derivative market on Injective, the oracleBase would be BTC/USD, the oracleQuote would be USDT/USD and the oracleType would be Chainlink. The price for the market would then be obtained by dividing the BTC/USD price with the USDT/USD price, leaving the BTC/USDT price.
