package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	exchangetypes "github.com/InjectiveLabs/sdk-go/chain/exchange/types"
)

type ExchangeSetupSuite struct {
	MarketID string
}

type Market struct {
	MarketID string `json:"market_id"`
}

type Response struct {
	Markets []Market `json:"markets"`
}

func extractMarketID(stdout []byte) (string, error) {
	var resp Response
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return "", fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	if len(resp.Markets) == 0 {
		return "", fmt.Errorf("no markets found")
	}
	return resp.Markets[0].MarketID, nil
}

func ExchangeSetup(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user1, user2 ibc.Wallet) ExchangeSetupSuite {
	userAddress := user1.FormattedAddress()
	quoteDenom := fmt.Sprintf("factory/%s/usdt", userAddress)

	msgCreateDenom := NewMsgCreateDenom(
		userAddress,
		"usdt",
		"Tether USD",
		"USDT",
		6,
		false,
	)
	MustBroadcastMsg(t, chain, ctx, user1, msgCreateDenom)

	msgMint := NewMsgMint(userAddress, quoteDenom)
	MustBroadcastMsg(t, chain, ctx, user1, msgMint)

	proposal := &exchangetypes.BatchExchangeModificationProposal{
		Title:       "Update Denom Min Notional",
		Description: "Set min notional for USDT",
		DenomMinNotionalProposal: &exchangetypes.DenomMinNotionalProposal{
			Title:       ".",
			Description: ".",
			DenomMinNotionals: []*exchangetypes.DenomMinNotional{
				{
					Denom:       quoteDenom,
					MinNotional: math.LegacyNewDec(1),
				},
			},
		},
	}
	msgProposal := &exchangetypes.MsgBatchExchangeModification{
		Sender:   authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Proposal: proposal,
	}
	MustSucceedProposal(t, chain, ctx, user1, msgProposal, "USDT Min Notional")

	msgInstantSpotMarketLaunch := &exchangetypes.MsgInstantSpotMarketLaunch{
		Sender:              userAddress,
		Ticker:              "INJ / USDT",
		BaseDenom:           "inj",
		QuoteDenom:          quoteDenom,
		MinPriceTickSize:    math.LegacyNewDecWithPrec(1, 4),
		MinQuantityTickSize: math.LegacyNewDecWithPrec(1, 4),
		MinNotional:         math.LegacyNewDec(1),
		BaseDecimals:        18,
		QuoteDecimals:       6,
	}
	MustBroadcastMsg(t, chain, ctx, user1, msgInstantSpotMarketLaunch)

	stdout, _, err := chain.Validators[0].ExecQuery(ctx, "exchange", "spot-markets", "--chain-id", chain.Config().ChainID)
	require.NoError(t, err, "error querying spot markets")

	tier1Stake := math.NewInt(100)
	tier2Stake := math.NewInt(1000)

	proposal = &exchangetypes.BatchExchangeModificationProposal{
		Title:       "Update Denom Min Notional",
		Description: "Set min notional for USDT",
		FeeDiscountProposal: &exchangetypes.FeeDiscountProposal{
			Title:       ".",
			Description: ".",
			Schedule: &exchangetypes.FeeDiscountSchedule{
				BucketCount:    2,
				BucketDuration: 30,
				QuoteDenoms:    []string{quoteDenom},
				TierInfos: []*exchangetypes.FeeDiscountTierInfo{
					{
						MakerDiscountRate: math.LegacyMustNewDecFromStr("0.1"),
						TakerDiscountRate: math.LegacyMustNewDecFromStr("0.1"),
						StakedAmount:      tier1Stake,
						Volume:            math.LegacyMustNewDecFromStr("0.3"),
					},
					{
						MakerDiscountRate: math.LegacyMustNewDecFromStr("0.3"),
						TakerDiscountRate: math.LegacyMustNewDecFromStr("0.3"),
						StakedAmount:      tier2Stake,
						Volume:            math.LegacyMustNewDecFromStr("3"),
					},
				},
				DisqualifiedMarketIds: []string{},
			},
		},
	}
	msgProposal = &exchangetypes.MsgBatchExchangeModification{
		Sender:   authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Proposal: proposal,
	}
	MustSucceedProposal(t, chain, ctx, user1, msgProposal, "Fee Discount Schedule")

	vals, err := chain.StakingQueryValidators(ctx, stakingtypes.Bonded.String())
	require.NoError(t, err)
	require.NotEmpty(t, vals)
	stakeMsg := &stakingtypes.MsgDelegate{
		DelegatorAddress: userAddress,
		ValidatorAddress: vals[0].OperatorAddress,
		Amount:           sdk.NewCoin(chain.Config().Denom, tier2Stake),
	}
	MustBroadcastMsg(t, chain, ctx, user1, stakeMsg)

	marketID, err := extractMarketID(stdout)
	require.NoError(t, err, "error parsing market id")

	defaultSubaccountUser1 := common.BytesToHash(common.RightPadBytes(user1.Address(), 32)).Hex()
	msgBuy := &exchangetypes.MsgCreateSpotLimitOrder{
		Sender: userAddress,
		Order: exchangetypes.SpotOrder{
			MarketId: marketID,
			OrderInfo: exchangetypes.OrderInfo{
				SubaccountId: defaultSubaccountUser1,
				FeeRecipient: "",
				Price:        math.LegacyNewDec(2),
				Quantity:     math.LegacyNewDec(100),
			},
			OrderType:    exchangetypes.OrderType_BUY,
			TriggerPrice: nil,
		},
	}
	MustBroadcastMsg(t, chain, ctx, user1, msgBuy)

	defaultSubaccountUser2 := common.BytesToHash(common.RightPadBytes(user2.Address(), 32)).Hex()
	msgSell := &exchangetypes.MsgCreateSpotLimitOrder{
		Sender: user2.FormattedAddress(),
		Order: exchangetypes.SpotOrder{
			MarketId: marketID,
			OrderInfo: exchangetypes.OrderInfo{
				SubaccountId: defaultSubaccountUser2,
				FeeRecipient: "",
				Price:        math.LegacyNewDec(2),
				Quantity:     math.LegacyNewDec(100),
			},
			OrderType:    exchangetypes.OrderType_SELL,
			TriggerPrice: nil,
		},
	}
	MustBroadcastMsg(t, chain, ctx, user2, msgSell)

	// wait for blocks to finalize
	time.Sleep(2 * time.Second)

	return ExchangeSetupSuite{
		MarketID: marketID,
	}
}
