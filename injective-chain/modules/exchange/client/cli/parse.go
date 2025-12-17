//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// orderParams holds the basic parameters extracted from command flags
type orderParams struct {
	marketId        string
	orderType       v2.OrderType
	reduceOnly      bool
	price           math.LegacyDec
	quantity        math.LegacyDec
	subaccountId    string
	feeRecipient    string
	expirationBlock int64
}

func parseSubmitFeeDiscountProposalFlags(fs *pflag.FlagSet) (*v2.FeeDiscountProposal, error) {
	proposal := &v2.FeeDiscountProposal{}
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

func parseBatchCommunityPoolSpendProposalFlags(fs *pflag.FlagSet) (*v2.BatchCommunityPoolSpendProposal, error) {
	proposal := &v2.BatchCommunityPoolSpendProposal{}
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

func parseTradingRewardCampaignUpdateProposalFlags(fs *pflag.FlagSet) (*v2.TradingRewardCampaignUpdateProposal, error) {
	proposal := &v2.TradingRewardCampaignUpdateProposal{}
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

func parseTradingRewardCampaignLaunchProposalFlags(fs *pflag.FlagSet) (*v2.TradingRewardCampaignLaunchProposal, error) {
	proposal := &v2.TradingRewardCampaignLaunchProposal{}
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

func parseTradingRewardPointsUpdateProposalFlags(fs *pflag.FlagSet) (*v2.TradingRewardPendingPointsUpdateProposal, error) {
	proposal := &v2.TradingRewardPendingPointsUpdateProposal{}
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

func parseBatchExchangeModificationsProposalFlags(fs *pflag.FlagSet) (*v2.BatchExchangeModificationProposal, error) {
	proposalFile, err := fs.GetString(govcli.FlagProposal)
	if err != nil {
		return nil, err
	}

	bz, err := os.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	var content v2.BatchExchangeModificationProposal

	um := jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}

	if err := um.Unmarshal(bytes.NewReader(bz), &content); err != nil {
		return nil, err
	}

	return &content, nil
}

func parseDerivativeOrderFlags(cmd *cobra.Command, ctx client.Context) (*v2.DerivativeOrder, error) {
	// Extract basic order parameters
	orderParams, err := extractBasicOrderParams(cmd)
	if err != nil {
		return nil, err
	}

	// Calculate margin if not a reduce-only order
	margin := math.LegacyZeroDec()
	if !orderParams.reduceOnly {
		margin, err = calculateOrderMargin(ctx, orderParams)
		if err != nil {
			return nil, err
		}
	}

	// Build the order
	order := buildDerivativeOrder(orderParams, margin)

	// Add optional CID if present
	if err := addOptionalCID(cmd, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// extractBasicOrderParams extracts the basic order parameters from command flags
func extractBasicOrderParams(cmd *cobra.Command) (*orderParams, error) {
	marketId, err := cmd.Flags().GetString(FlagMarketID)
	if err != nil {
		return nil, err
	}

	orderType, err := orderTypeFromFlag(cmd, FlagOrderType)
	if err != nil {
		return nil, err
	}

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

	subaccountId, err := cmd.Flags().GetString(FlagSubaccountID)
	if err != nil {
		return nil, err
	}

	feeRecipient, err := cmd.Flags().GetString(FlagFeeRecipient)
	if err != nil {
		return nil, err
	}

	expirationBlock, err := cmd.Flags().GetString(FlagExpirationBlock)
	if err != nil {
		expirationBlock = "0"
	}

	expirationBlockInt, err := strconv.ParseInt(expirationBlock, 10, 64)
	if err != nil {
		return nil, err
	}

	return &orderParams{
		marketId:        marketId,
		orderType:       orderType,
		reduceOnly:      reduceOnly,
		price:           price,
		quantity:        quantity,
		subaccountId:    subaccountId,
		feeRecipient:    feeRecipient,
		expirationBlock: expirationBlockInt,
	}, nil
}

// calculateOrderMargin calculates the margin for a non-reduce-only order
func calculateOrderMargin(ctx client.Context, params *orderParams) (math.LegacyDec, error) {
	// Get the market information
	market, err := findMarketById(ctx, params.marketId)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	// Calculate margin based on order type
	return calculateMarginAmount(params, market)
}

// findMarketById retrieves the market information by its ID
func findMarketById(ctx client.Context, marketId string) (*v2.BinaryOptionsMarket, error) {
	queryClient := v2.NewQueryClient(ctx)
	req := &v2.QueryBinaryMarketsRequest{
		Status: "Active",
	}

	response, err := queryClient.BinaryOptionsMarkets(context.Background(), req)
	if err != nil {
		return nil, err
	}

	for _, m := range response.GetMarkets() {
		if m.MarketId == marketId {
			return m, nil
		}
	}

	return nil, fmt.Errorf("cannot find market with id: %s", marketId)
}

// calculateMarginAmount calculates the margin amount based on order parameters and market information
func calculateMarginAmount(params *orderParams, market *v2.BinaryOptionsMarket) (math.LegacyDec, error) {
	if params.orderType == v2.OrderType_BUY {
		return params.price.Mul(params.quantity), nil
	}
	return types.GetScaledPrice(math.LegacyOneDec(), market.OracleScaleFactor).Sub(params.price).Mul(params.quantity), nil
}

// buildDerivativeOrder creates a derivative order from the given parameters
func buildDerivativeOrder(params *orderParams, margin math.LegacyDec) v2.DerivativeOrder {
	return v2.DerivativeOrder{
		MarketId: params.marketId,
		OrderInfo: v2.OrderInfo{
			SubaccountId: params.subaccountId,
			FeeRecipient: params.feeRecipient,
			Price:        params.price,
			Quantity:     params.quantity,
		},
		OrderType:       params.orderType,
		Margin:          margin,
		TriggerPrice:    nil, // not supported currently
		ExpirationBlock: params.expirationBlock,
	}
}

// addOptionalCID adds the CID to the order if it's present in the flags
func addOptionalCID(cmd *cobra.Command, order *v2.DerivativeOrder) error {
	cidFlag := cmd.Flags().Lookup(FlagCID)
	if cidFlag != nil {
		cid, err := cmd.Flags().GetString(FlagCID)
		if err != nil {
			return err
		}
		order.OrderInfo.Cid = cid
	}
	return nil
}

func orderTypeFromFlag(cmd *cobra.Command, flag string) (v2.OrderType, error) {
	orderTypeStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return v2.OrderType_UNSPECIFIED, err
	}

	var orderType v2.OrderType
	switch orderTypeStr {
	case "buy":
		orderType = v2.OrderType_BUY
	case "buyPostOnly":
		orderType = v2.OrderType_BUY_PO
	case "sell":
		orderType = v2.OrderType_SELL
	case "sellPostOnly":
		orderType = v2.OrderType_SELL_PO
	case "buyAtomic":
		orderType = v2.OrderType_BUY_ATOMIC
	case "sellAtomic":
		orderType = v2.OrderType_SELL_ATOMIC
	default:
		return v2.OrderType_UNSPECIFIED, errors.New("order type must be \"buy\", \"buyPostOnly\", \"sellPostOnly\" or \"sell\" or \"buyAtomic\" or \"sellAtomic\"")
	}

	return orderType, nil
}

func marketStatusFromFlag(cmd *cobra.Command, flag string) (v2.MarketStatus, error) {
	marketStatusStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return v2.MarketStatus_Unspecified, err
	}

	var marketStatus v2.MarketStatus
	switch marketStatusStr {
	case "active":
		marketStatus = v2.MarketStatus_Active
	case "paused":
		marketStatus = v2.MarketStatus_Paused
	case "demolished":
		marketStatus = v2.MarketStatus_Demolished
	case "expired":
		marketStatus = v2.MarketStatus_Expired
	default:
		marketStatus = v2.MarketStatus_Unspecified
	}
	return marketStatus, nil
}
