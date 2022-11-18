---
sidebar_position: 5
title: Events
---
# Events

The oracle module emits the following events:
## Band
### Msg/RelayBandRates
| Type              | Attribute Key | Attribute Value |
| ----------------- | ------------- | --------------- |
| SetBandPriceEvent | module        | oracle          |
| SetBandPriceEvent | relayer       | {relayer}       |
| SetBandPriceEvent | symbol        | {symbol}        |
| SetBandPriceEvent | price         | {price}         |
| SetBandPriceEvent | resolve_time  | {resolve_time}  |
| SetBandPriceEvent | request_id    | {request_id}    |

## Msg/MsgRequestBandIBCRates

| Type        | Attribute Key            | Attribute Value      |
| ----------- | ------------------------ | -------------------- |
| send_packet | packet_data              | {data}               |
| send_packet | packet_timeout_height    | {timeoutHeight}      |
| send_packet | packet_timeout_timestamp | {timeoutTimestamp}   |
| send_packet | packet_sequence          | {sequence}           |
| send_packet | packet_src_port          | {sourcePort}         |
| send_packet | packet_src_channel       | {sourceChannel}      |
| send_packet | packet_dst_port          | {destinationPort}    |
| send_packet | packet_dst_channel       | {destinationChannel} |
| send_packet | packet_channel_ordering  | {channel.Ordering}   |
| message     | action                   | requestBandIBCRates  |
| message     | module                   | oracle               |

## Coinbase

### Msg/RelayCoinbaseMessages
| Type                  | Attribute Key | Attribute Value |
| --------------------- | ------------- | --------------- |
| SetCoinbasePriceEvent | module        | oracle          |
| SetCoinbasePriceEvent | symbol        | {symbol}        |
| SetCoinbasePriceEvent | price         | {price}         |
| SetCoinbasePriceEvent | timestamp     | {timestamp}     |


## PriceFeed
### Msg/RelayPriceFeedPrice
| Type                   | Attribute Key | Attribute Value |
| ---------------------- | ------------- | --------------- |
| SetPriceFeedPriceEvent | module        | oracle          |
| SetPriceFeedPriceEvent | relayer       | {relayer}       |
| SetPriceFeedPriceEvent | base          | {base}          |
| SetPriceFeedPriceEvent | quote         | {quote}         |
