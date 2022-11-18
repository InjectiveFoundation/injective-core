---
sidebar_position: 7
title: Events
---

# Events

The ocr module emits the following events:

## Handlers

### MsgCreateFeed

| Type    | Attribute Key | Attribute Value |
| ------- | ------------- | --------------- |
| message | action        | MsgCreateFeed   |
| message | sender        | {sender}        |

### MsgUpdateFeed

| Type    | Attribute Key | Attribute Value |
| ------- | ------------- | --------------- |
| message | action        | MsgUpdateFeed   |
| message | sender        | {sender}        |

### MsgTransmit

| Type                 | Attribute Key         | Attribute Value         |
| -------------------- | --------------------- | ----------------------- |
| EventNewTransmission | FeedId                | {FeedId}                |
| EventNewTransmission | AggregatorRoundId     | {AggregatorRoundId}     |
| EventNewTransmission | Answer                | {Answer}                |
| EventNewTransmission | Transmitter           | {Transmitter}           |
| EventNewTransmission | ObservationsTimestamp | {ObservationsTimestamp} |
| EventNewTransmission | Observations          | {Observations}          |
| EventNewTransmission | Observers             | {Observers}             |
| EventNewTransmission | ConfigDigest          | {ConfigDigest}          |
| EventNewTransmission | EpochAndRound         | {EpochAndRound}         |
| EventTransmitted     | ConfigDigest          | {ConfigDigest}          |
| EventTransmitted     | Epoch                 | {Epoch}                 |
| message              | action                | MsgTransmit             |
| message              | sender                | {sender}                |

### MsgFundFeedRewardPool

| Type    | Attribute Key | Attribute Value       |
| ------- | ------------- | --------------------- |
| message | action        | MsgFundFeedRewardPool |
| message | sender        | {sender}              |

### MsgWithdrawFeedRewardPool

| Type    | Attribute Key | Attribute Value           |
| ------- | ------------- | ------------------------- |
| message | action        | MsgWithdrawFeedRewardPool |
| message | sender        | {sender}                  |

### MsgSetPayees

| Type    | Attribute Key | Attribute Value |
| ------- | ------------- | --------------- |
| message | action        | MsgSetPayees    |
| message | sender        | {sender}        |

### MsgTransferPayeeship

| Type    | Attribute Key | Attribute Value      |
| ------- | ------------- | -------------------- |
| message | action        | MsgTransferPayeeship |
| message | sender        | {sender}             |

### MsgAcceptPayeeship

| Type    | Attribute Key | Attribute Value    |
| ------- | ------------- | ------------------ |
| message | action        | MsgAcceptPayeeship |
| message | sender        | {sender}           |

## Proposals

### SetConfigProposal

| Type           | Attribute Key             | Attribute Value             |
| -------------- | ------------------------- | --------------------------- |
| EventConfigSet | ConfigDigest              | {ConfigDigest}              |
| EventConfigSet | PreviousConfigBlockNumber | {PreviousConfigBlockNumber} |
| EventConfigSet | Config                    | {Config}                    |
| EventConfigSet | ConfigInfo                | {ConfigInfo}                |

### SetBatchConfigProposal

| Type             | Attribute Key             | Attribute Value             |
| ---------------- | ------------------------- | --------------------------- |
| EventConfigSet[] | ConfigDigest              | {ConfigDigest}              |
| EventConfigSet[] | PreviousConfigBlockNumber | {PreviousConfigBlockNumber} |
| EventConfigSet[] | Config                    | {Config}                    |
| EventConfigSet[] | ConfigInfo                | {ConfigInfo}                |

## BeginBlocker

| Type | Attribute Key | Attribute Value |
| ---- | ------------- | --------------- |
