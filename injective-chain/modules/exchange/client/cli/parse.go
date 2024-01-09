//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func parseSubmitFeeDiscountProposalFlags(fs *pflag.FlagSet) (*exchangetypes.FeeDiscountProposal, error) {
	proposal := &exchangetypes.FeeDiscountProposal{}
	proposalFile, _ := fs.GetString(govcli.FlagProposal)

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func parseBatchCommunityPoolSpendProposalFlags(fs *pflag.FlagSet) (*exchangetypes.BatchCommunityPoolSpendProposal, error) {
	proposal := &exchangetypes.BatchCommunityPoolSpendProposal{}
	proposalFile, _ := fs.GetString(govcli.FlagProposal)

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func parseTradingRewardCampaignUpdateProposalFlags(fs *pflag.FlagSet) (*exchangetypes.TradingRewardCampaignUpdateProposal, error) {
	proposal := &exchangetypes.TradingRewardCampaignUpdateProposal{}
	proposalFile, _ := fs.GetString(govcli.FlagProposal)

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func parseTradingRewardCampaignLaunchProposalFlags(fs *pflag.FlagSet) (*exchangetypes.TradingRewardCampaignLaunchProposal, error) {
	proposal := &exchangetypes.TradingRewardCampaignLaunchProposal{}
	proposalFile, _ := fs.GetString(govcli.FlagProposal)

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func parseTradingRewardPointsUpdateProposalFlags(fs *pflag.FlagSet) (*exchangetypes.TradingRewardPendingPointsUpdateProposal, error) {
	proposal := &exchangetypes.TradingRewardPendingPointsUpdateProposal{}
	proposalFile, _ := fs.GetString(govcli.FlagProposal)

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func parseBatchExchangeModificationsProposalFlags(fs *pflag.FlagSet) (*exchangetypes.BatchExchangeModificationProposal, error) {
	proposalFile, err := fs.GetString(govcli.FlagProposal)
	if err != nil {
		return nil, err
	}

	bz, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	content := exchangetypes.BatchExchangeModificationProposal{}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bz))
	jsonDecoder.DisallowUnknownFields()

	err = jsonDecoder.Decode(&content)
	if err != nil {
		return nil, err
	}

	return &content, nil
}

func parseDerivativeOrderFlags(cmd *cobra.Command, ctx client.Context) (*exchangetypes.DerivativeOrder, error) {

	marketId, err := cmd.Flags().GetString(FlagMarketID)
	if err != nil {
		return nil, err
	}

	queryClient := exchangetypes.NewQueryClient(ctx)
	req := &exchangetypes.QueryBinaryMarketsRequest{
		Status: "Active",
	}

	orderType, err := orderTypeFromFlag(cmd, FlagOrderType)
	if err != nil {
		return nil, err
	}

	margin := sdk.ZeroDec()

	reduceOnly, err := cmd.Flags().GetBool(FlagReduceOnly)
	if err != nil {
		return nil, err
	}

	price, err := decimalFromFlag(cmd, FlagPrice)
	if err != nil {
		return nil, err
	}

	quantity, err := decimalFromFlag(cmd, FlagQuantity)
	if err != nil {
		return nil, err
	}

	if !reduceOnly {
		response, err := queryClient.BinaryOptionsMarkets(context.Background(), req)
		if err != nil {
			return nil, err
		}
		var market *exchangetypes.BinaryOptionsMarket
		for _, m := range response.GetMarkets() {
			if m.MarketId == marketId {
				market = m
			}
		}
		if market == nil {
			return nil, errors.New(fmt.Sprintf("Cannot find market with id: %s", marketId))
		}

		if orderType == exchangetypes.OrderType_BUY {
			margin = price.Mul(quantity)
		} else {
			margin = exchangetypes.GetScaledPrice(sdk.OneDec(), market.OracleScaleFactor).Sub(price).Mul(quantity)
		}
	}
	subaccountId, err := cmd.Flags().GetString(FlagSubaccountID)
	if err != nil {
		return nil, err
	}

	feeRecipient, err := cmd.Flags().GetString(FlagFeeRecipient)
	if err != nil {
		return nil, err
	}

	order := exchangetypes.DerivativeOrder{
		MarketId: marketId,
		OrderInfo: exchangetypes.OrderInfo{
			SubaccountId: subaccountId,
			FeeRecipient: feeRecipient,
			Price:        price,
			Quantity:     quantity,
		},
		OrderType:    orderType,
		Margin:       margin,
		TriggerPrice: nil, // not supported currently
	}
	return &order, nil
}

func orderTypeFromFlag(cmd *cobra.Command, flag string) (exchangetypes.OrderType, error) {
	orderTypeStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return exchangetypes.OrderType_UNSPECIFIED, err
	}

	var orderType exchangetypes.OrderType
	switch orderTypeStr {
	case "buy":
		orderType = exchangetypes.OrderType_BUY
	case "buyPostOnly":
		orderType = exchangetypes.OrderType_BUY_PO
	case "sell":
		orderType = exchangetypes.OrderType_SELL
	case "sellPostOnly":
		orderType = exchangetypes.OrderType_SELL_PO
	default:
		return exchangetypes.OrderType_UNSPECIFIED, errors.New("order type must be \"buy\", \"buyPostOnly\", \"sellPostOnly\" or \"sell\"")
	}

	return orderType, nil
}

func marketStatusFromFlag(cmd *cobra.Command, flag string) (exchangetypes.MarketStatus, error) {
	marketStatusStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return exchangetypes.MarketStatus_Unspecified, err
	}

	var marketStatus exchangetypes.MarketStatus
	switch marketStatusStr {
	case "active":
		marketStatus = exchangetypes.MarketStatus_Active
	case "paused":
		marketStatus = exchangetypes.MarketStatus_Paused
	case "demolished":
		marketStatus = exchangetypes.MarketStatus_Demolished
	case "expired":
		marketStatus = exchangetypes.MarketStatus_Expired
	default:
		marketStatus = exchangetypes.MarketStatus_Unspecified
	}
	return marketStatus, nil
}
