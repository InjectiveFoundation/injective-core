---
sidebar_position: 2
title: State
---

# State

Genesis state defines the initial state of the module to be used to setup the module.

```go
// GenesisState defines the OCR module's genesis state.
type GenesisState struct {
	// params defines all the parameters of related to OCR.
	Params Params 
	// feed_configs stores all of the supported OCR feeds
	FeedConfigs []*FeedConfig
	// latest_epoch_and_rounds stores the latest epoch and round for each feedId
	LatestEpochAndRounds []*FeedEpochAndRound
	// feed_transmissions stores the last transmission for each feed
	FeedTransmissions []*FeedTransmission
	// latest_aggregator_round_ids stores the latest aggregator round ID for each feedId
	LatestAggregatorRoundIds []*FeedLatestAggregatorRoundIDs
	// reward_pools stores the reward pools
	RewardPools []*RewardPool
	// feed_observation_counts stores the feed observation counts
	FeedObservationCounts []*FeedCounts
	// feed_transmission_counts stores the feed transmission counts
	FeedTransmissionCounts []*FeedCounts
	// pending_payeeships stores the pending payeeships
	PendingPayeeships []*PendingPayeeship
}
```
## Params

`Params` is a module-wide configuration that stores system parameters and defines overall functioning of the ocr module.
This module is modifiable by governance using params update proposal natively supported by `gov` module.

Struct for the `ocr` module params store.
```go
type Params struct {
	// Native denom for LINK coin in the bank keeper
	LinkDenom string
	// The block number interval at which payouts are made
	PayoutBlockInterval uint64
	// The admin for the OCR module
	ModuleAdmin string
}
```

## FeedConfig

`FeedConfig` is to manage the configurations of feed and it exists one per feed.

```go
type FeedConfig struct {
	// signers ith element is address ith oracle uses to sign a report
	Signers []string
	// transmitters ith element is address ith oracle uses to transmit a report via the transmit method
	Transmitters []string
	// f maximum number of faulty/dishonest oracles the protocol can tolerate while still working correctly
	F uint32
	// onchain_config contains properties relevant only for the Cosmos module.
	OnchainConfig *OnchainConfig
	// offchain_config_version version of the serialization format used for "offchain_config" parameter
	OffchainConfigVersion uint64
	// offchain_config serialized data used by oracles to configure their offchain operation
	OffchainConfig []byte
}
```

### FeedConfigInfo

`FeedConfigInfo` is storing the information that needs to be updated more often for each transmission event.

```go
type FeedConfigInfo struct {
	LatestConfigDigest []byte
	F                  uint32
	N                  uint32
	// config_count ordinal number of this config setting among all config settings
	ConfigCount             uint64
	LatestConfigBlockNumber int64
}
```

### Transmission

`Transmission` is the unit to save transition information on the store.

```go
// Transmission records the median answer from the transmit transaction at
// time timestamp
type Transmission struct {
	Answer                math.LegacyDec
	ObservationsTimestamp int64
	TransmissionTimestamp int64
}
```

### Report

`Report` is the unit to save report information on the store.

```go
type Report struct {
	ObservationsTimestamp int64
	Observers             []byte
	Observations          []math.LegacyDec
}
```

`ReportToSign` saves the information that needs to be signed by observers.

```go
type ReportToSign struct {
	ConfigDigest []byte 
	Epoch        uint64
	Round        uint64 
	ExtraHash    []byte
	// Opaque report
	Report []byte
}
```

### OnchainConfig

`OnchainConfig` saves the configuration that needs to be managed on-chain for feed config.

```go
type OnchainConfig struct {
	// chain_id the ID of the Cosmos chain itself.
	ChainId string
	// feed_id is an unique ID for the target of this config
	FeedId string
	// lowest answer the median of a report is allowed to be
	MinAnswer math.LegacyDec
	// highest answer the median of a report is allowed to be
	MaxAnswer math.LegacyDec
	// Fixed LINK reward for each observer
	LinkPerObservation math.Int
	// Fixed LINK reward for transmitter
	LinkPerTransmission math.Int
	// Native denom for LINK coin in the bank keeper
	LinkDenom string
	// Enables unique reports
	UniqueReports bool
	// short human-readable description of observable this feed's answers pertain to
	Description string
	// feed administrator
	FeedAdmin string
	// feed billing administrator
	BillingAdmin string
}
```

### ContractConfig

`ContractConfig` saves the configuration that is related to contract to store OCR.

```go
type ContractConfig struct {
	// config_count ordinal number of this config setting among all config settings
	ConfigCount uint64
	// signers ith element is address ith oracle uses to sign a report
	Signers []string 
	// transmitters ith element is address ith oracle uses to transmit a report via the transmit method
	Transmitters []string
	// f maximum number of faulty/dishonest oracles the protocol can tolerate while still working correctly
	F uint32
	// onchain_config serialized config that is relevant only for the module.
	OnchainConfig []byte
	// offchain_config_version version of the serialization format used for "offchain_config" parameter
	OffchainConfigVersion uint64
	// offchain_config serialized data used by oracles to configure their offchain operation
	OffchainConfig []byte
}
```
### FeedProperties

`FeedProperties` is a unit to store the properties of feed by id.

```go
type FeedProperties struct {
	// feed_id is an unique ID for the target of this config
	FeedId string
	// f maximum number of faulty/dishonest oracles the protocol can tolerate while still working correctly
	F uint32
	// offchain_config_version version of the serialization format used for "offchain_config" parameter
	OffchainConfigVersion uint64
	// offchain_config serialized data used by oracles to configure their offchain operation
	OffchainConfig []byte
	// lowest answer the median of a report is allowed to be
	MinAnswer math.LegacyDec
	// highest answer the median of a report is allowed to be
	MaxAnswer math.LegacyDec
	// Fixed LINK reward for each observer
	LinkPerObservation math.Int
	// Fixed LINK reward for transmitter
	LinkPerTransmission math.Int
	// Enables unique reports
	UniqueReports bool
	// short human-readable description of observable this feed's answers pertain to
	Description string
}
```

### PendingPayeeship

`PendingPayeeship` is a record that is stored when a person is delegating payeeship to another address.
When proposed payee accept this, this record is removed.

```go
type PendingPayeeship struct {
	FeedId        string
	Transmitter   string
	ProposedPayee string
}
```