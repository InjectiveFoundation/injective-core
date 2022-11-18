package keeper

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type OcrConfig interface {
	SetFeedConfig(
		ctx sdk.Context,
		feedId string,
		config *types.FeedConfig,
	) (prevInfo *types.FeedConfigInfo)

	GetFeedConfig(
		ctx sdk.Context,
		feedId string,
	) *types.FeedConfig

	GetFeedConfigByDigest(
		ctx sdk.Context,
		configDigest []byte,
	) *types.FeedConfig

	SetFeedConfigInfo(
		ctx sdk.Context,
		feedId string,
		configInfo *types.FeedConfigInfo,
	)

	GetFeedConfigInfo(
		ctx sdk.Context,
		feedId string,
	) *types.FeedConfigInfo

	IsTransmitter(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
	) bool

	GetAllTransmitters(
		ctx sdk.Context,
		feedId string,
	) []string

	IsSigner(
		ctx sdk.Context,
		feedId string,
		signer sdk.AccAddress,
	) bool

	GetAllSigners(
		ctx sdk.Context,
		feedId string,
	) []string

	GetAllFeedConfigs(
		ctx sdk.Context,
	) []*types.FeedConfig
}

func (k *keeper) SetFeedConfig(
	ctx sdk.Context,
	feedId string,
	feedConfig *types.FeedConfig,
) (prevInfo *types.FeedConfigInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	prevInfo = k.GetFeedConfigInfo(ctx, feedId)
	if prevInfo == nil {
		prevInfo = &types.FeedConfigInfo{}
	}

	key := types.GetFeedConfigKey(feedId)
	bz := k.cdc.MustMarshal(feedConfig)
	k.getStore(ctx).Set(key, bz)

	cc := &types.ContractConfig{
		ConfigCount:           prevInfo.ConfigCount + 1,
		Signers:               feedConfig.Signers,
		Transmitters:          feedConfig.Transmitters,
		F:                     feedConfig.F,
		OnchainConfig:         feedConfig.OnchainConfig,
		OffchainConfigVersion: feedConfig.OffchainConfigVersion,
		OffchainConfig:        feedConfig.OffchainConfig,
	}

	configDigest := cc.Digest(ctx.ChainID(), feedConfig.ModuleParams.FeedId)

	configInfo := &types.FeedConfigInfo{
		LatestConfigDigest:      configDigest,
		F:                       feedConfig.F,
		N:                       uint32(len(feedConfig.Signers)),
		ConfigCount:             prevInfo.ConfigCount + 1,
		LatestConfigBlockNumber: ctx.BlockHeight(),
	}
	k.SetFeedConfigInfo(ctx, feedId, configInfo)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConfigSet{
		ConfigDigest:              configDigest,
		PreviousConfigBlockNumber: prevInfo.LatestConfigBlockNumber,
		Config:                    feedConfig,
		ConfigInfo:                configInfo,
	})

	if k.hooks != nil {
		k.hooks.AfterSetFeedConfig(ctx, feedConfig)
	}

	return prevInfo
}

func (k *keeper) GetFeedConfig(
	ctx sdk.Context,
	feedId string,
) *types.FeedConfig {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetFeedConfigKey(feedId))
	if bz == nil {
		return nil
	}

	var feedConfig types.FeedConfig
	k.cdc.MustUnmarshal(bz, &feedConfig)
	return &feedConfig
}

func (k *keeper) GetFeedConfigByDigest(
	ctx sdk.Context,
	configDigest []byte,
) *types.FeedConfig {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	feedConfigStore := prefix.NewStore(k.getStore(ctx), types.FeedConfigInfoPrefix)
	iterator := feedConfigStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var feedConfigInfo types.FeedConfigInfo
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &feedConfigInfo)

		if bytes.Equal(feedConfigInfo.LatestConfigDigest, configDigest) {
			// TODO: ensure Key there is without prefix
			feedIdBz := iterator.Key()
			feedId := types.GetFeedIdFromPaddedFeedIdBz(feedIdBz)
			feedConfig := k.GetFeedConfig(ctx, feedId)
			if feedConfig != nil {
				return feedConfig
			}
		}
	}

	return nil
}

func (k *keeper) SetFeedConfigInfo(
	ctx sdk.Context,
	feedId string,
	configInfo *types.FeedConfigInfo,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetFeedConfigInfoKey(feedId)
	bz := k.cdc.MustMarshal(configInfo)
	k.getStore(ctx).Set(key, bz)
}

func (k *keeper) GetFeedConfigInfo(
	ctx sdk.Context,
	feedId string,
) *types.FeedConfigInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetFeedConfigInfoKey(feedId))
	if bz == nil {
		return nil
	}

	var feedConfigInfo types.FeedConfigInfo
	k.cdc.MustUnmarshal(bz, &feedConfigInfo)
	return &feedConfigInfo
}

func (k *keeper) IsTransmitter(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	transmitterStr := transmitter.String()

	cfg := k.GetFeedConfig(ctx, feedId)
	if cfg != nil {
		for _, t := range cfg.Transmitters {
			if t == transmitterStr {
				return true
			}
		}
	}

	return false
}

func (k *keeper) GetAllTransmitters(
	ctx sdk.Context,
	feedId string,
) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	cfg := k.GetFeedConfig(ctx, feedId)
	if cfg != nil {
		return cfg.Transmitters
	}

	return []string{}
}

func (k *keeper) IsSigner(
	ctx sdk.Context,
	feedId string,
	signer sdk.AccAddress,
) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	signerStr := signer.String()

	cfg := k.GetFeedConfig(ctx, feedId)
	if cfg != nil {
		for _, t := range cfg.Signers {
			if t == signerStr {
				return true
			}
		}
	}

	return false
}

func (k *keeper) GetAllSigners(
	ctx sdk.Context,
	feedId string,
) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	cfg := k.GetFeedConfig(ctx, feedId)
	if cfg != nil {
		return cfg.Signers
	}

	return []string{}
}

func (k *keeper) GetAllFeedConfigs(
	ctx sdk.Context,
) []*types.FeedConfig {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	feedConfigs := make([]*types.FeedConfig, 0, 32)
	store := ctx.KVStore(k.storeKey)
	feedConfigStore := prefix.NewStore(store, types.FeedConfigPrefix)

	iterator := feedConfigStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var feedConfig types.FeedConfig
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &feedConfig)
		feedConfigs = append(feedConfigs, &feedConfig)
	}

	return feedConfigs
}
