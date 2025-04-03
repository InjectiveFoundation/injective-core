//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzcli "github.com/cosmos/cosmos-sdk/x/authz/client/cli"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govgeneraltypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/version"
)

// NewTxCmd returns a root CLI command handler for certain modules/exchange transaction commands.
func NewTxCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, false)
	cmd.AddCommand(
		// market admin
		NewUpdateSpotMarketCmd(),
		NewUpdateDerivativeMarketCmd(),
		// spot markets
		NewInstantSpotMarketLaunchTxCmd(),
		NewCreateSpotLimitOrderTxCmd(),
		NewCreateSpotMarketOrderTxCmd(),
		NewCancelSpotLimitOrderTxCmd(),
		// perp markets
		NewInstantPerpetualMarketLaunchTxCmd(),
		NewCreateDerivativeLimitOrderTxCmd(),
		NewCreateDerivativeMarketOrderTxCmd(),
		NewCancelDerivativeLimitOrderTxCmd(),
		// expiry futures
		NewInstantExpiryFuturesMarketLaunchTxCmd(),
		NewExpiryFuturesMarketLaunchProposalTxCmd(),
		// binary options markets
		NewInstantBinaryOptionsMarketLaunchTxCmd(),
		NewAdminUpdateBinaryOptionsMarketTxCmd(),
		NewCreateBinaryOptionsMarketOrderTxCmd(),
		NewCreateBinaryOptionsLimitOrderTxCmd(),
		NewCancelBinaryOptionsOrderTxCmd(),
		// proposals
		NewSpotMarketLaunchProposalTxCmd(),
		NewSpotMarketUpdateParamsProposalTxCmd(),
		NewPerpetualMarketLaunchProposalTxCmd(),
		NewDerivativeMarketParamUpdateProposalTxCmd(),
		NewBatchExchangeModificationProposalTxCmd(),
		TradingRewardCampaignLaunchProposalTxCmd(),
		TradingRewardCampaignUpdateProposalTxCmd(),
		TradingRewardPointsUpdateProposalTxCmd(),
		FeeDiscountProposalTxCmd(),
		BatchCommunityPoolSpendProposalTxCmd(),
		NewAtomicMarketOrderFeeMultiplierScheduleProposalTxCmd(),
		// account
		NewDepositTxCmd(),
		NewWithdrawTxCmd(),
		NewSubaccountTransferTxCmd(),
		NewExternalTransferTxCmd(),
		NewRewardsOptOutTxCmd(),
		// mito
		NewSubscribeToSpotVaultTxCmd(),
		NewRedeemFromSpotVaultTxCmd(),
		NewRedeemFromAmmVaultTxCmd(),
		NewSubscribeToDerivativeVaultTxCmd(),
		NewRedeemFromDerivativeVaultTxCmd(),
		NewPrivilegedExecuteContractTxCmd(),
		NewSubscribeToAmmVaultTxCmd(),
		// authz
		NewAuthzTxCmd(),
		NewBatchUpdateAuthzTxCmd(),
		// stake grant
		NewStakeGrantAuthorizationTxCmd(),
		NewStakeGrantActivationTxCmd(),
		// other
		NewExchangeEnableProposalTxCmd(),
		NewMarketForcedSettlementTxCmd(),
		NewUpdateDenomDecimalsProposalTxCmd(),
		NewIncreasePositionMarginTxCmd(),
		NewDecreasePositionMarginTxCmd(),
		NewMsgLiquidatePositionTxCmd(),
	)
	return cmd
}

func NewInstantSpotMarketLaunchTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"instant-spot-market-launch <ticker> <base_denom> <quote_denom>",
		"Launch spot market by paying listing fee without governance",
		&types.MsgInstantSpotMarketLaunch{},
		cli.FlagsMapping{
			"MinPriceTickSize":    cli.Flag{Flag: FlagMinPriceTickSize, UseDefaultIfOmitted: true},
			"MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize, UseDefaultIfOmitted: true},
			"MinNotional":         cli.Flag{Flag: FlagMinNotional, UseDefaultIfOmitted: true},
			"BaseDecimals":        cli.Flag{Flag: FlagBaseDecimals},
			"QuoteDecimals":       cli.Flag{Flag: FlagQuoteDecimals},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = `tx exchange instant-spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=1000000000 \
			--min-quantity-tick-size=1000000000000000 \
			--min-notional=1 \
			--base-decimals=18 \
			--quote-decimals=6`
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagBaseDecimals, "0", "base token decimals")
	cmd.Flags().String(FlagQuoteDecimals, "0", "quote token decimals")
	return cmd
}

func NewCreateSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-limit-order <order_type> <market_ticker> <quantity> <price> <client_order_id>",
		"Create Spot Limit Order",
		&types.MsgCreateSpotLimitOrder{},
		cli.FlagsMapping{"TriggerPrice": cli.SkipField}, // disable parsing of trigger price
		cli.ArgsMapping{
			"OrderType": cli.Arg{
				Index: 0,
				Transform: func(orig string, ctx grpc.ClientConn) (any, error) {
					var orderType types.OrderType
					switch orig {
					case "buy":
						orderType = types.OrderType_BUY
					case "sell":
						orderType = types.OrderType_SELL
					case "buy-PO":
						orderType = types.OrderType_BUY_PO
					case "sell-PO":
						orderType = types.OrderType_SELL_PO
					default:
						return orderType, fmt.Errorf(
							`order type must be "buy", "sell", "buy-PO" or "sell-PO"`,
						)
					}
					return int(orderType), nil
				},
			},
			"MarketId": cli.Arg{Index: 1, Transform: getSpotMarketIdFromTicker},
			"Price":    cli.Arg{Index: 3},
			"Quantity": cli.Arg{Index: 2},
			"Cid":      cli.Arg{Index: 4},
		},
	)
	cmd.Example = "injectived tx exchange create-spot-limit-order buy ETH/USDT 2.4 2000.1 my_order_1 --from=genesis --keyring-backend=file --yes"
	return cmd
}

func NewCreateSpotMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-market-order <order_type> <market_ticker> <quantity> <worst_price> <client_order_id>",
		"Create Spot Market Order",
		&types.MsgCreateSpotMarketOrder{},
		cli.FlagsMapping{"TriggerPrice": cli.SkipField}, // disable parsing of trigger price
		cli.ArgsMapping{
			"OrderType": cli.Arg{
				Index: 0,
				Transform: func(orig string, ctx grpc.ClientConn) (any, error) {
					var orderType types.OrderType
					switch orig {
					case "buy":
						orderType = types.OrderType_BUY
					case "sell":
						orderType = types.OrderType_SELL
					default:
						return orderType, fmt.Errorf(`order type must be "buy", "sell"`)
					}
					return int(orderType), nil
				},
			},
			"MarketId": cli.Arg{Index: 1, Transform: getSpotMarketIdFromTicker},
			"Price":    cli.Arg{Index: 3},
			"Quantity": cli.Arg{Index: 2},
			"Cid":      cli.Arg{Index: 4},
		},
	)
	cmd.Example = "injectived tx exchange create-spot-limit-order buy ETH/USDT 2.4 2000.1 my_order_1 --from=genesis --keyring-backend=file --yes"
	return cmd
}

func NewCancelSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-spot-limit-order",
		"Cancel Spot Limit Order",
		&types.MsgCancelSpotOrder{},
		cli.FlagsMapping{
			"MarketId": cli.Flag{
				Flag:      FlagMarketID,
				Transform: getSpotMarketIdFromTicker,
			},
			"OrderHash": cli.Flag{Flag: FlagOrderHash, UseDefaultIfOmitted: true},
			"Cid":       cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = "injectived tx exchange cancel-spot-limit-order --market-id=ETH/USDT --order-hash=0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8"
	cmd.Flags().String(FlagMarketID, "", "Spot market ID")
	cmd.Flags().String(FlagOrderHash, "", "Order hash")
	cmd.Flags().String(FlagCID, "", "Client order ID")
	return cmd
}

func NewCancelDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-derivative-limit-order",
		"Cancel Derivative Limit Order",
		&types.MsgCancelDerivativeOrder{},
		cli.FlagsMapping{
			"OrderMask": cli.SkipField,
			"MarketId": cli.Flag{
				Flag:      FlagMarketID,
				Transform: getDerivativeMarketIdFromTicker,
			},
			"OrderHash": cli.Flag{Flag: FlagOrderHash, UseDefaultIfOmitted: true},
			"Cid":       cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = "tx exchange cancel-derivative-limit-order --market-id=ETH/USDT --order-hash=0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8"
	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderHash, "", "Order hash")
	cmd.Flags().String(FlagCID, "", "Client order ID")
	return cmd
}

func NewCreateDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-derivative-limit-order",
		"Create Derivative Limit Order",
		&types.MsgCreateDerivativeLimitOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.Flag{Flag: FlagTriggerPrice},
			"OrderType": cli.Flag{
				Flag: FlagOrderType,
				Transform: func(orig string, ctx grpc.ClientConn) (any, error) {
					var orderType types.OrderType
					switch orig {
					case "buy":
						orderType = types.OrderType_BUY
					case "sell":
						orderType = types.OrderType_SELL
					case "buy-PO":
						orderType = types.OrderType_BUY_PO
					case "sell-PO":
						orderType = types.OrderType_SELL_PO
					case "take-sell":
						orderType = types.OrderType_TAKE_SELL
					case "stop-sell":
						orderType = types.OrderType_STOP_SELL
					case "stop-buy":
						orderType = types.OrderType_STOP_BUY
					case "take-buy":
						orderType = types.OrderType_TAKE_BUY
					default:
						return orderType, fmt.Errorf(
							`order type must be "buy", "sell", "take-sell", "stop-sell", "take-buy", "stop-buy", "buy-PO" or "sell-PO"`,
						)
					}
					return int(orderType), nil
				},
			},
			"MarketId": cli.Flag{
				Flag:      FlagMarketID,
				Transform: getDerivativeMarketIdFromTicker,
			},
			"Price":        cli.Flag{Flag: FlagPrice},
			"Quantity":     cli.Flag{Flag: FlagQuantity},
			"Margin":       cli.Flag{Flag: FlagMargin},
			"SubaccountId": cli.Flag{Flag: FlagSubaccountID},
			"Cid":          cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = `injectived tx exchange create-derivative-limit-order \
			--order-type="buy" \
			--market-id="ETH/USDT" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--price="4.1" \
			--quantity="10.01" \
			--margin="30.0" \
			--cid="my_order_1" \
			--from=genesis \
			--keyring-backend=file \
			--yes`
	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	cmd.Flags().String(FlagCID, "", "Client order ID")
	cmd.Flags().String(FlagTriggerPrice, "0", "Trigger price")
	return cmd
}

func NewCreateDerivativeMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-derivative-market-order",
		"Create Derivative Market Order",
		&types.MsgCreateDerivativeMarketOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.Flag{Flag: FlagTriggerPrice},
			"OrderType": cli.Flag{
				Flag: FlagOrderType,
				Transform: func(orig string, ctx grpc.ClientConn) (any, error) {
					var orderType types.OrderType
					switch orig {
					case "buy":
						orderType = types.OrderType_BUY
					case "sell":
						orderType = types.OrderType_SELL
					case "take-sell":
						orderType = types.OrderType_TAKE_SELL
					case "stop-sell":
						orderType = types.OrderType_STOP_SELL
					case "stop-buy":
						orderType = types.OrderType_STOP_BUY
					case "take-buy":
						orderType = types.OrderType_TAKE_BUY
					case "buy-atomic":
						orderType = types.OrderType_BUY_ATOMIC
					case "sell-atomic":
						orderType = types.OrderType_SELL_ATOMIC
					default:
						return orderType, fmt.Errorf(`order type must be "buy", "sell", "take-buy", "stop-buy", "take-sell" or "stop-sell" or "buy-atomic" or "sell-atomic"`)
					}
					return int(orderType), nil
				},
			},
			"MarketId": cli.Flag{
				Flag:      FlagMarketID,
				Transform: getDerivativeMarketIdFromTicker,
			},
			"Price":        cli.Flag{Flag: FlagPrice},
			"Quantity":     cli.Flag{Flag: FlagQuantity},
			"Margin":       cli.Flag{Flag: FlagMargin},
			"SubaccountId": cli.Flag{Flag: FlagSubaccountID},
			"Cid":          cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = `tx exchange create-derivative-market-order \
			--order-type="buy" \
			--market-id="ETH/USDT" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--price="4.1" \
			--quantity="10.01" \
			--margin="30.0" \
			--cid="my_order_1"`
	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	cmd.Flags().String(FlagCID, "", "Client order ID")
	cmd.Flags().String(FlagTriggerPrice, "0", "Trigger price")
	return cmd
}

func NewSpotMarketUpdateParamsProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.SpotMarketParamUpdateProposal{
		AdminInfo: &types.AdminInfo{},
	})
	cmd := cli.TxCmd(
		"update-spot-market-params",
		"Submit a proposal to update spot market params",
		proposalMsgDummy, cli.FlagsMapping{
			"Title":               cli.Flag{Flag: govcli.FlagTitle},
			"Description":         cli.Flag{Flag: govcli.FlagDescription},
			"MarketId":            cli.Flag{Flag: FlagMarketID},
			"Ticker":              cli.Flag{Flag: FlagTicker},
			"MakerFeeRate":        cli.Flag{Flag: FlagMakerFeeRate},
			"TakerFeeRate":        cli.Flag{Flag: FlagTakerFeeRate},
			"RelayerFeeShareRate": cli.Flag{Flag: FlagRelayerFeeShareRate},
			"MinPriceTickSize":    cli.Flag{Flag: FlagMinPriceTickSize},
			"MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize},
			"MinNotional":         cli.Flag{Flag: FlagMinNotional},
			"Admin":               cli.Flag{Flag: FlagAdmin},
			"AdminPermissions":    cli.Flag{Flag: FlagAdminPermissions},
			"Status": cli.Flag{
				Flag: FlagMarketStatus,
				Transform: func(origV string, ctx grpc.ClientConn) (tranformedV any, err error) {
					var status types.MarketStatus
					if origV != "" {
						if newStatus, ok := types.MarketStatus_value[origV]; ok {
							status = types.MarketStatus(newStatus)
						} else {
							return nil, fmt.Errorf("incorrect market status: %s", origV)
						}
					} else {
						status = types.MarketStatus_Unspecified
					}
					return fmt.Sprintf("%v", int32(status)), nil
				},
			},
			"BaseDecimals":   cli.Flag{Flag: FlagBaseDecimals},
			"QuoteDecimals":  cli.Flag{Flag: FlagQuoteDecimals},
			"InitialDeposit": cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange update-spot-market-params \
			--market-id="0xacdd4f9cb90ecf5c4e254acbf65a942f562ca33ba718737a93e5cb3caadec3aa" \
			--base-decimals=18 \
			--quote-decimals=6 \
			--title="Spot market params update" \
			--description="XX" \
			--deposit="1000000000000000000inj"`

	cmd.Flags().String(FlagMarketID, "", "Spot market ID")
	cmd.Flags().String(FlagTicker, "", "market ticker")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagRelayerFeeShareRate, "", "relayer fee share rate")
	cmd.Flags().String(FlagMinPriceTickSize, "", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagMarketStatus, "", "market status")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")
	cmd.Flags().Uint32(FlagBaseDecimals, 0, "base asset decimals")
	cmd.Flags().Uint32(FlagQuoteDecimals, 0, "quote asset decimals")
	cliflags.AddGovProposalFlags(cmd)

	return cmd
}

func NewSpotMarketLaunchProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.SpotMarketLaunchProposal{
		AdminInfo: &types.AdminInfo{},
	})
	cmd := cli.TxCmd(
		"spot-market-launch <ticker> <base_denom> <quote_denom>",
		"Submit a proposal to launch spot-market",
		proposalMsgDummy,
		cli.FlagsMapping{
			"Title":               cli.Flag{Flag: govcli.FlagTitle},
			"Description":         cli.Flag{Flag: govcli.FlagDescription},
			"MakerFeeRate":        cli.Flag{Flag: FlagMakerFeeRate},
			"TakerFeeRate":        cli.Flag{Flag: FlagTakerFeeRate},
			"MinPriceTickSize":    cli.Flag{Flag: FlagMinPriceTickSize},
			"MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize},
			"MinNotional":         cli.Flag{Flag: FlagMinNotional},
			"Admin":               cli.Flag{Flag: FlagAdmin},
			"AdminPermissions":    cli.Flag{Flag: FlagAdminPermissions},
			"BaseDecimals":        cli.Flag{Flag: FlagBaseDecimals},
			"QuoteDecimals":       cli.Flag{Flag: FlagQuoteDecimals},
			"InitialDeposit":      cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=1000000000 \
			--min-quantity-tick-size=1000000000000000 \
			--min-notional=1000000000 \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--base-decimals=18 \
			--quote-decimals=6 \
			--title="INJ/ATOM spot market" \
			--description="XX" \
			--deposit="1000000000000000000inj"`

	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")
	cmd.Flags().String(FlagBaseDecimals, "0", "base token decimals")
	cmd.Flags().String(FlagQuoteDecimals, "0", "quote token decimals")
	cliflags.AddGovProposalFlags(cmd)

	return cmd
}

func NewExchangeEnableProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.ExchangeEnableProposal{})
	cmd := cli.TxCmd(
		"propose-exchange-enable <exchange-type>",
		"Submit a proposal to enable spot or derivatives exchange (exchangeType of spot or derivatives)",
		proposalMsgDummy,
		cli.FlagsMapping{
			"Title":          cli.Flag{Flag: govcli.FlagTitle},
			"Description":    cli.Flag{Flag: govcli.FlagDescription},
			"InitialDeposit": cli.Flag{Flag: govcli.FlagDeposit},
		},
		cli.ArgsMapping{
			"ExchangeType": cli.Arg{
				Index: 0,
				Transform: func(origV string, ctx grpc.ClientConn) (tranformedV any, err error) {
					var exchangeType types.ExchangeType
					switch origV {
					case "spot":
						exchangeType = types.ExchangeType_SPOT
					case "derivatives":
						exchangeType = types.ExchangeType_DERIVATIVES
					default:
						return nil, fmt.Errorf("incorrect exchange type %s", origV)
					}
					return fmt.Sprintf("%v", int32(exchangeType)), nil
				},
			},
		},
	)
	cmd.Example = `tx exchange spot --title="Enable Spot Exchange" --description="Enable Spot Exchange" --deposit="1000000000000000000inj"`
	cliflags.AddGovProposalFlags(cmd)
	return cmd
}

func NewPerpetualMarketLaunchProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.PerpetualMarketLaunchProposal{
		AdminInfo: &types.AdminInfo{},
	})
	cmd := cli.TxCmd(
		"propose-perpetual-market",
		"Submit a proposal to launch perpetual market",
		proposalMsgDummy,
		cli.FlagsMapping{
			"Title":             cli.Flag{Flag: govcli.FlagTitle},
			"Description":       cli.Flag{Flag: govcli.FlagDescription},
			"Ticker":            cli.Flag{Flag: FlagTicker},
			"QuoteDenom":        cli.Flag{Flag: FlagQuoteDenom},
			"OracleBase":        cli.Flag{Flag: FlagOracleBase},
			"OracleQuote":       cli.Flag{Flag: FlagOracleQuote},
			"OracleScaleFactor": cli.Flag{Flag: FlagOracleScaleFactor},
			"OracleType": cli.Flag{
				Flag: FlagOracleType,
				Transform: func(origV string, ctx grpc.ClientConn) (tranformedV any, err error) {
					if oracleType, err := oracletypes.GetOracleType(origV); err != nil {
						return nil, fmt.Errorf("error parsing oracle type: %w", err)
					} else {
						return fmt.Sprintf("%v", int32(oracleType)), nil
					}
				},
			},
			"InitialMarginRatio":     cli.Flag{Flag: FlagInitialMarginRatio},
			"MaintenanceMarginRatio": cli.Flag{Flag: FlagMaintenanceMarginRatio},
			"MakerFeeRate":           cli.Flag{Flag: FlagMakerFeeRate},
			"TakerFeeRate":           cli.Flag{Flag: FlagTakerFeeRate},
			"MinPriceTickSize":       cli.Flag{Flag: FlagMinPriceTickSize},
			"MinQuantityTickSize":    cli.Flag{Flag: FlagMinQuantityTickSize},
			"MinNotional":            cli.Flag{Flag: FlagMinNotional},
			"Admin":                  cli.Flag{Flag: FlagAdmin},
			"AdminPermissions":       cli.Flag{Flag: FlagAdminPermissions},
			"InitialDeposit":         cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange propose-perpetual-market
			--ticker="INJ/USDT" \
			--quote-denom="usdt" \
			--oracle-base="inj" \
			--oracle-quote="usdt" \
			--oracle-type="pricefeed" \
			--oracle-scale-factor="0" \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--initial-margin-ratio="0.05" \
			--maintenance-margin-ratio="0.02" \
			--min-price-tick-size="0.0001" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1000000000" \
			--title="INJ perpetual market" \
			--description="XX" \
			--deposit="1000000000000000000inj"`
	cmd.Flags().String(FlagTicker, "", "ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "quote denom")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")

	cliflags.AddGovProposalFlags(cmd)
	return cmd
}

func NewExpiryFuturesMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-expiry-futures-market [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to launch expiry futures market",
		Long: `Submit a proposal to launch expiry futures market.

		Example:
		$ %s tx exchange propose-expiry-futures-market \
			--ticker="INJ/USDT-0625" \
			--quote-denom="usdt" \
			--oracle-base="inj" \
			--oracle-quote="usdt-0625" \
			--oracle-type="pricefeed" \
			--oracle-scale-factor="0" \
			--expiry="1624586400" \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--initial-margin-ratio="0.05" \
			--maintenance-margin-ratio="0.02" \
			--min-price-tick-size="0.0001" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1" \
			--title="INJ/ATOM expiry futures market" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			quoteDenom, err := cmd.Flags().GetString(FlagQuoteDenom)
			if err != nil {
				return err
			}
			oracleBase, err := cmd.Flags().GetString(FlagOracleBase)
			if err != nil {
				return err
			}
			oracleQuote, err := cmd.Flags().GetString(FlagOracleQuote)
			if err != nil {
				return err
			}
			oracleTypeStr, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeStr)
			if err != nil {
				return err
			}

			oracleScaleFactor, err := cmd.Flags().GetUint32(FlagOracleScaleFactor)
			if err != nil {
				return err
			}

			expiry, err := cmd.Flags().GetInt64(FlagExpiry)
			if err != nil {
				return err
			}

			initialMarginRatio, err := decimalFromFlag(cmd, FlagInitialMarginRatio)
			if err != nil {
				return err
			}

			maintenanceMarginRatio, err := decimalFromFlag(cmd, FlagMaintenanceMarginRatio)
			if err != nil {
				return err
			}

			makerFeeRate, err := decimalFromFlag(cmd, FlagMakerFeeRate)
			if err != nil {
				return err
			}

			takerFeeRate, err := decimalFromFlag(cmd, FlagTakerFeeRate)
			if err != nil {
				return err
			}

			minPriceTickSizeStr, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}
			minQuantityTickSizeStr, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}
			minNotionalStr, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			minPriceTickSize, err := math.LegacyNewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}
			minQuantityTickSize, err := math.LegacyNewDecFromStr(minQuantityTickSizeStr)
			if err != nil {
				return err
			}
			minNotional, err := math.LegacyNewDecFromStr(minNotionalStr)
			if err != nil {
				return err
			}

			content, err := expiryFuturesMarketLaunchArgsToContent(
				cmd,
				ticker,
				quoteDenom,
				oracleBase,
				oracleQuote,
				oracleScaleFactor,
				oracleType,
				expiry,
				initialMarginRatio,
				maintenanceMarginRatio,
				makerFeeRate,
				takerFeeRate,
				minPriceTickSize,
				minQuantityTickSize,
				minNotional,
			)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "quote denom")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().Int64(FlagExpiry, -1, "initial margin ratio")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewInstantPerpetualMarketLaunchTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instant-perpetual-market-launch [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Instantly launch perpetual market by paying listing fee.",
		Long: `Instantly launch perpetual market by paying listing fee.

		Example:
		$ %s tx exchange instant-perpetual-market-launch \
			--ticker="INJ/USDT-0625" \
			--quote-denom="usdt" \
			--oracle-base="inj" \
			--oracle-quote="usdt" \
			--oracle-type="pricefeed" \
			--oracle-scale-factor="0" \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--initial-margin-ratio="0.05" \
			--maintenance-margin-ratio="0.02" \
			--min-price-tick-size="0.0001" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1000000" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			quoteDenom, err := cmd.Flags().GetString(FlagQuoteDenom)
			if err != nil {
				return err
			}
			oracleBase, err := cmd.Flags().GetString(FlagOracleBase)
			if err != nil {
				return err
			}
			oracleQuote, err := cmd.Flags().GetString(FlagOracleQuote)
			if err != nil {
				return err
			}
			oracleTypeStr, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeStr)
			if err != nil {
				return err
			}

			oracleScaleFactor, err := cmd.Flags().GetUint32(FlagOracleScaleFactor)
			if err != nil {
				return err
			}

			initialMarginRatio, err := decimalFromFlag(cmd, FlagInitialMarginRatio)
			if err != nil {
				return err
			}

			maintenanceMarginRatio, err := decimalFromFlag(cmd, FlagMaintenanceMarginRatio)
			if err != nil {
				return err
			}

			makerFeeRate, err := decimalFromFlag(cmd, FlagMakerFeeRate)
			if err != nil {
				return err
			}

			takerFeeRate, err := decimalFromFlag(cmd, FlagTakerFeeRate)
			if err != nil {
				return err
			}

			minPriceTickSizeStr, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}
			minQuantityTickSizeStr, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}
			minNotionalString, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			minPriceTickSize, err := math.LegacyNewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := math.LegacyNewDecFromStr(minQuantityTickSizeStr)
			if err != nil {
				return err
			}

			minNotional, err := math.LegacyNewDecFromStr(minNotionalString)
			if err != nil {
				return err
			}

			msg := &types.MsgInstantPerpetualMarketLaunch{
				Sender:                 clientCtx.GetFromAddress().String(),
				Ticker:                 ticker,
				QuoteDenom:             quoteDenom,
				OracleBase:             oracleBase,
				OracleQuote:            oracleQuote,
				OracleScaleFactor:      oracleScaleFactor,
				OracleType:             oracleType,
				MakerFeeRate:           makerFeeRate,
				TakerFeeRate:           takerFeeRate,
				InitialMarginRatio:     initialMarginRatio,
				MaintenanceMarginRatio: maintenanceMarginRatio,
				MinPriceTickSize:       minPriceTickSize,
				MinQuantityTickSize:    minQuantityTickSize,
				MinNotional:            minNotional,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "quote denom")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewInstantBinaryOptionsMarketLaunchTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instant-binary-options-market-launch [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Instantly launch a binary options market by paying a listing fee.",
		Long: `Instantly launch a binary options market by paying a listing fee.

		Example:
		$ %s tx exchange instant-binary-options-market-launch \
			--ticker="UFC-KHABIB-TKO-05/30/2023" \
			--quote-denom="peggy0xdAC17F958D2ee523a2206206994597C13D831ec7" \
			--oracle-symbol="UFC-KHABIB-TKO-05/30/2023" \
			--oracle-provider="ufc" \
			--oracle-type="provider" \
			--oracle-scale-factor="6" \
			--maker-fee-rate="0.0005" \
			--taker-fee-rate="0.0012" \
			--expiry="1685460582" \
			--settlement-time="1690730982" \
			--min-price-tick-size="10000" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			quoteDenom, err := cmd.Flags().GetString(FlagQuoteDenom)
			if err != nil {
				return err
			}
			oracleSymbol, err := cmd.Flags().GetString(FlagOracleSymbol)
			if err != nil {
				return err
			}
			oracleProvider, err := cmd.Flags().GetString(FlagOracleProvider)
			if err != nil {
				return err
			}
			oracleTypeStr, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeStr)
			if err != nil {
				return err
			}

			oracleScaleFactor, err := cmd.Flags().GetUint32(FlagOracleScaleFactor)
			if err != nil {
				return err
			}

			admin, err := cmd.Flags().GetString(FlagAdmin)
			if err != nil {
				return err
			}

			makerFeeRate, err := decimalFromFlag(cmd, FlagMakerFeeRate)
			if err != nil {
				return err
			}

			takerFeeRate, err := decimalFromFlag(cmd, FlagTakerFeeRate)
			if err != nil {
				return err
			}

			expiry, err := cmd.Flags().GetInt64(FlagExpiry)
			if err != nil {
				return err
			}

			settlementTime, err := cmd.Flags().GetInt64(FlagSettlementTime)
			if err != nil {
				return err
			}

			minPriceTickSizeStr, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}
			minQuantityTickSizeStr, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}
			minNotionalStr, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			minPriceTickSize, err := math.LegacyNewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}
			minQuantityTickSize, err := math.LegacyNewDecFromStr(minQuantityTickSizeStr)
			if err != nil {
				return err
			}
			minNotional, err := math.LegacyNewDecFromStr(minNotionalStr)
			if err != nil {
				return err
			}

			msg := &types.MsgInstantBinaryOptionsMarketLaunch{
				Sender:              clientCtx.GetFromAddress().String(),
				Ticker:              ticker,
				OracleSymbol:        oracleSymbol,
				OracleProvider:      oracleProvider,
				OracleType:          oracleType,
				OracleScaleFactor:   oracleScaleFactor,
				MakerFeeRate:        makerFeeRate,
				TakerFeeRate:        takerFeeRate,
				ExpirationTimestamp: expiry,
				SettlementTimestamp: settlementTime,
				Admin:               admin,
				QuoteDenom:          quoteDenom,
				MinPriceTickSize:    minPriceTickSize,
				MinQuantityTickSize: minQuantityTickSize,
				MinNotional:         minNotional,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "quote denom")
	cmd.Flags().String(FlagOracleSymbol, "", "oracle symbol")
	cmd.Flags().String(FlagOracleProvider, "", "oracle provider")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().Int64(FlagExpiry, 0, "expiration UNIX timestamp seconds")
	cmd.Flags().Int64(FlagSettlementTime, 0, "settlement UNIX timestamp seconds")
	cmd.Flags().String(FlagAdmin, "", "admin of the market")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewAdminUpdateBinaryOptionsMarketTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin-update-binary-options-market [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Updates binary options market.",
		Long: `Updates binary options market.

		Example:
		$ %s tx exchange admin-update-binary-options-market \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--settlement-price="10000.0" \
			--settlement-time="1685460582" \
			--expiration-time="1685460582" \
			--market-status="active" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			marketId, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			var settlementPrice *math.LegacyDec
			settlementPriceIn, err := decimalFromFlag(cmd, FlagSettlementPrice)
			if err == nil {
				settlementPrice = &settlementPriceIn
			}

			expirationTimestamp, err := cmd.Flags().GetInt64(FlagExpirationTime)
			if err != nil {
				return err
			}

			settlementTime, err := cmd.Flags().GetInt64(FlagSettlementTime)
			if err != nil {
				return err
			}

			marketStatus, err := marketStatusFromFlag(cmd, FlagMarketStatus)
			if err != nil {
				return err
			}

			msg := &types.MsgAdminUpdateBinaryOptionsMarket{
				Sender:              clientCtx.GetFromAddress().String(),
				MarketId:            marketId,
				SettlementPrice:     settlementPrice,
				ExpirationTimestamp: expirationTimestamp,
				SettlementTimestamp: settlementTime,
				Status:              marketStatus,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketID, "", "market id")
	cmd.Flags().String(FlagSettlementPrice, "", "settlement price")
	cmd.Flags().Int64(FlagExpirationTime, 0, "Expiration time")
	cmd.Flags().Int64(FlagSettlementTime, 0, "settlement time")
	cmd.Flags().String(FlagMarketStatus, "", "market status")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCreateBinaryOptionsLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-binary-options-limit-order [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Creates binary options limit order.",
		Long: `Creates binary options limit order.

		Example:
		$ %s tx exchange create-binary-options-limit-order \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--fee-recipient="" \
			--price="1.0" \
			--quantity="10.01" \
			--order-type="buy" \
			--margin="30.0" \
			--cid="my_order_1" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			order, err := parseDerivativeOrderFlags(cmd, clientCtx)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateBinaryOptionsLimitOrder{
				Sender: clientCtx.GetFromAddress().String(),
				Order:  *order,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	defineDerivativeOrderFlags(cmd)
	return cmd
}

func NewCreateBinaryOptionsMarketOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-binary-options-market-order [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Creates binary options market order.",
		Long: `Creates binary options market order.

		Example:
		$ %s tx exchange create-binary-options-market-order \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--fee-recipient="" \
			--price="1.0" \
			--quantity="10.01" \
			--order-type="buy" \
			--margin="30.0" \
			--trigger-price="10.0" \
			--cid="my_order_1" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			order, err := parseDerivativeOrderFlags(cmd, clientCtx)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateBinaryOptionsMarketOrder{
				Sender: clientCtx.GetFromAddress().String(),
				Order:  *order,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	defineDerivativeOrderFlags(cmd)
	return cmd
}

func NewCancelBinaryOptionsOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-binary-options-order [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Creates binary options market order.",
		Long: `Creates binary options market order.

		Example:
		$ %s tx exchange cancel-binary-options-order \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--order-hash="1F55C246EC6616C64A241E77D503B58BD2A5AA1966D713398FF070214CDAEFA5" \ '
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cancelMessage := types.MsgCancelBinaryOptionsOrder{}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cancelMessage.Sender = clientCtx.GetFromAddress().String()

			marketId, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}
			cancelMessage.MarketId = marketId

			subaccountId, err := cmd.Flags().GetString(FlagSubaccountID)
			if err != nil {
				return err
			}
			cancelMessage.SubaccountId = subaccountId

			orderHashFlag := cmd.Flags().Lookup(FlagOrderHash)
			if orderHashFlag != nil {
				orderHash, err := cmd.Flags().GetString(FlagOrderHash)
				if err != nil {
					return err
				}
				cancelMessage.OrderHash = orderHash
			}

			cidFlag := cmd.Flags().Lookup(FlagCID)
			if cidFlag != nil {
				cid, err := cmd.Flags().GetString(FlagCID)
				if err != nil {
					return err
				}
				cancelMessage.Cid = cid
			}

			msg := &cancelMessage

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketID, "", "market id")
	cmd.Flags().String(FlagSubaccountID, "", "subaccount id")
	cmd.Flags().String(FlagOrderHash, "", "order (trasnasction) hash")
	cmd.Flags().String(FlagCID, "", "client order id")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewInstantExpiryFuturesMarketLaunchTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instant-expiry-futures-market-launch [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Instantly launch expiry futures market by paying listing fee",
		Long: `Instantly launch expiry futures market by paying listing fee.

		Example:
		$ %s tx exchange instant-expiry-futures-market-launch \
			--ticker="INJ/USDT-0625" \
			--quote-denom="usdt" \
			--oracle-base="inj" \
			--oracle-quote="usdt-0625" \
			--oracle-type="pricefeed" \
			--oracle-scale-factor="0" \
			--expiry="1624586400" \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--initial-margin-ratio="0.05" \
			--maintenance-margin-ratio="0.02" \
			--min-price-tick-size="0.0001" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			quoteDenom, err := cmd.Flags().GetString(FlagQuoteDenom)
			if err != nil {
				return err
			}
			oracleBase, err := cmd.Flags().GetString(FlagOracleBase)
			if err != nil {
				return err
			}
			oracleQuote, err := cmd.Flags().GetString(FlagOracleQuote)
			if err != nil {
				return err
			}
			oracleTypeStr, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeStr)
			if err != nil {
				return err
			}

			oracleScaleFactor, err := cmd.Flags().GetUint32(FlagOracleScaleFactor)
			if err != nil {
				return err
			}

			expiry, err := cmd.Flags().GetInt64(FlagExpiry)
			if err != nil {
				return err
			}

			initialMarginRatio, err := decimalFromFlag(cmd, FlagInitialMarginRatio)
			if err != nil {
				return err
			}

			maintenanceMarginRatio, err := decimalFromFlag(cmd, FlagMaintenanceMarginRatio)
			if err != nil {
				return err
			}

			makerFeeRate, err := decimalFromFlag(cmd, FlagMakerFeeRate)
			if err != nil {
				return err
			}

			takerFeeRate, err := decimalFromFlag(cmd, FlagTakerFeeRate)
			if err != nil {
				return err
			}

			minPriceTickSizeStr, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}
			minQuantityTickSizeStr, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}
			minNotionalStr, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			minPriceTickSize, err := math.LegacyNewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}
			minQuantityTickSize, err := math.LegacyNewDecFromStr(minQuantityTickSizeStr)
			if err != nil {
				return err
			}
			minNotional, err := math.LegacyNewDecFromStr(minNotionalStr)
			if err != nil {
				return err
			}

			msg := &types.MsgInstantExpiryFuturesMarketLaunch{
				Sender:                 clientCtx.GetFromAddress().String(),
				Ticker:                 ticker,
				QuoteDenom:             quoteDenom,
				OracleBase:             oracleBase,
				OracleQuote:            oracleQuote,
				OracleType:             oracleType,
				OracleScaleFactor:      oracleScaleFactor,
				Expiry:                 expiry,
				MakerFeeRate:           makerFeeRate,
				TakerFeeRate:           takerFeeRate,
				InitialMarginRatio:     initialMarginRatio,
				MaintenanceMarginRatio: maintenanceMarginRatio,
				MinPriceTickSize:       minPriceTickSize,
				MinQuantityTickSize:    minQuantityTickSize,
				MinNotional:            minNotional,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "quote denom")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().Int64(FlagExpiry, -1, "expiry")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func TradingRewardCampaignLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trading-reward-campaign-launch-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to create a new trade rewards campaign",
		Long: `Submit a proposal to create a new trade rewards campaign.

		Example:
		$ %s tx exchange trading-reward-campaign-launch-proposal \
		    --proposal="path/to/trading-reward-campaign-launch-proposal.json"
			--from=genesis \
			--keyring-backend=file \
			--yes

			Where proposal.json contains:
			{
				"title": "title",
				"description": "description",
				"campaign_info": {
					"campaign_duration_seconds": 30000,
					"quote_denoms": [
						"quoteDenom1",
						"quoteDenom2"
					],
					"trading_reward_boost_info": {
					"spot_market_multipliers": null,
					"boosted_derivative_market_ids": [
						"marketID"
					],
					"derivative_market_multipliers": [
						{
							"maker_points_multiplier": "1.000000000000000000",
							"taker_points_multiplier": "3.000000000000000000"
						}
					]
					},
					"disqualified_market_ids": [
						"marketID1",
						"marketID2"
					]
				},
				"campaign_reward_pools": [
					{
						"start_timestamp": 100,
						"max_campaign_rewards": [
							{
								"denom": "inj",
								"amount": "1000000"
							}
						]
					}
				]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := parseTradingRewardCampaignLaunchProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func TradingRewardCampaignUpdateProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trading-reward-campaign-update-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to create trading reward campaign update.",
		Long: `Submit a proposal to create trading reward campaign update.

		Example:
		$ %s tx exchange trading-reward-campaign-update-proposal \
		    --proposal="path/to/trading-reward-campaign-update-proposal.json"
			--from=genesis \
			--keyring-backend=file \
			--yes

			Where proposal.json contains:
			{
				"title": "title",
				"description": "description",
				"campaign_info": {
					"campaign_duration_seconds": 30000,
					"quote_denoms": [
						"quoteDenom1",
						"quoteDenom2"
					],
					"trading_reward_boost_info": {
					"spot_market_multipliers": null,
					"boosted_derivative_market_ids": [
						"marketID"
					],
					"derivative_market_multipliers": [
						{
							"maker_points_multiplier": "1.000000000000000000",
							"taker_points_multiplier": "3.000000000000000000"
						}
					]
					},
					"disqualified_market_ids": [
						"marketID1",
						"marketID2"
					]
				}
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := parseTradingRewardCampaignUpdateProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func TradingRewardPointsUpdateProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trading-reward-points-update-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to update account trading rewards points.",
		Long: `Submit a proposal to update account trading rewards points.

		Example:
		$ %s tx exchange trading-reward-points-update-proposal \
		    --proposal="path/to/trading-reward-points-update-proposal.json"
			--from=genesis \
			--keyring-backend=file \
			--yes

			Where proposal.json contains:
			{
				"title": "title",
				"description": "description",
				"reward_point_updates": [
					{
						"account_address": "inj1wfawuv6fslzjlfa4v7exv27mk6rpfeyvhvxchc",
						"new_points": "150.000000000000000000"
					}
				]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := parseTradingRewardPointsUpdateProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func BatchCommunityPoolSpendProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-community-pool-spend-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to batch spend community pool.",
		Long: `Submit a proposal to batch spend community pool.

		Example:
		$ %s tx exchange batch-community-pool-spend-proposal \
		    --proposal="path/to/batch-community-pool-spend-proposal.json" \
			--from=genesis \
			--deposit="1000000000000000000inj" \
			--keyring-backend=file \
			--yes

			Where proposal.json contains:
{
  "title": "title",
  "description": "description",
  "proposals": [
    {
      "title": "title",
      "description": "description",
      "recipient": "inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz",
      "amount": [
        {
          "denom": "inj",
          "amount": "1000000"
        }
      ]
    }
  ]
}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := parseBatchCommunityPoolSpendProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func FeeDiscountProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fee-discount-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to create a new fee discount proposal",
		Long: `Submit a proposal to create a new fee discount proposal.

		Example:
		$ %s tx exchange fee-discount-proposal \
		    --proposal="path/to/fee-discount-proposal.json" \
			--from=genesis \
			--deposit="1000000000000000000inj" \
			--keyring-backend=file \
			--yes

			Where proposal.json contains:
			{
			  "title": "Fee Discount Proposal",
			  "description": "My awesome fee discount proposal",
			  "schedule": [
				{
				  "bucketCount": 30,
				  "bucketDuration": 30,
				  "quoteDenoms": [
					"peggy0xdAC17F958D2ee523a2206206994597C13D831ec7"
				  ],
				  "tierInfos": [
					{
					  "makerDiscountRate": "0.01",
					  "takerDiscountRate": "0.01",
					  "stakedAmount": "1000000000000000000",
					  "volume": "1000000"
					},
					{
					  "makerDiscountRate": "0.02",
					  "takerDiscountRate": "0.02",
					  "stakedAmount": "2000000000000000000",
					  "volume": "2000000"
					}
				  ],
				  "disqualifiedMarketIds": []
				}
			  ]
			}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := parseSubmitFeeDiscountProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewBatchExchangeModificationProposalTxCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "batch-exchange-modifications-proposal",
		Short: "Submit a proposal for batch exchange modifications",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a proposal for batch exchange modifications.
Example:
$ %s tx gov batch-exchange-modifications-proposal --proposal="path/to/proposal.json" --from=mykey --deposit=100000000000000000000inj --expedited=false

Where proposal.json contains:
{
  "title": "title",
  "description": "description",
  "spot_market_param_update_proposals": [
    {
      "title": "title",
      "description": "description",
      "market_id": "marketId",
      "maker_fee_rate": "0.020000000000000000",
      "taker_fee_rate": "0.020000000000000000",
      "relayer_fee_share_rate": "0.050000000000000000",
      "min_price_tick_size": "0.000010000000000000",
      "min_quantity_tick_size": "0.000010000000000000",
      "status": 1
    }
  ],
  "derivative_market_param_update_proposals": [
    {
      "title": "Update derivative market param",
      "description": "Update derivative market description",
      "market_id": "marketId",
      "initial_margin_ratio": "0.000010000000000000",
      "maintenance_margin_ratio": "0.000010000000000000",
      "maker_fee_rate": "0.020000000000000000",
      "taker_fee_rate": "0.020000000000000000",
      "relayer_fee_share_rate": "0.050000000000000000",
      "min_price_tick_size": "0.000010000000000000",
      "min_quantity_tick_size": "0.000010000000000000",
      "HourlyInterestRate": "0.000010000000000000",
      "HourlyFundingRateCap": "0.000010000000000000"
    }
  ],
  "spot_market_launch_proposals": [
    {
      "title": "Just a Title",
      "description": "Just a Description",
      "ticker": "ticker",
      "base_denom": "baseDenom",
      "quote_denom": "quoteDenom",
      "min_price_tick_size": "0.000100000000000000",
      "min_quantity_tick_size": "0.000100000000000000"
    }
  ],
  "perpetual_market_launch_proposals": [
    {
      "title": "Just a Title",
      "description": "Just a Description",
      "ticker": "ticker",
      "quote_denom": "quoteDenom",
      "oracle_base": "oracleBase",
      "oracle_quote": "oracleQuote",
      "oracle_type": 1,
      "initial_margin_ratio": "0.050000000000000000",
      "maintenance_margin_ratio": "0.020000000000000000",
      "maker_fee_rate": "0.001000000000000000",
      "taker_fee_rate": "0.001500000000000000",
      "min_price_tick_size": "0.000100000000000000",
      "min_quantity_tick_size": "0.000100000000000000"
    }
  ],
  "expiry_futures_market_launch_proposals": [
    {
      "title": "Just a Title",
      "description": "Just a Description",
      "ticker": "ticker",
      "quote_denom": "quoteDenom",
      "oracle_base": "oracleBase",
      "oracle_quote": "oracleQuote",
      "oracle_type": 1,
      "expiry": 1000,
      "initial_margin_ratio": "0.050000000000000000",
      "maintenance_margin_ratio": "0.020000000000000000",
      "maker_fee_rate": "0.001000000000000000",
      "taker_fee_rate": "0.001500000000000000",
      "min_price_tick_size": "0.000100000000000000",
      "min_quantity_tick_size": "0.000100000000000000"
    }
  ],
  "trading_reward_campaign_update_proposal": {
    "title": "Trade Reward Campaign",
    "description": "Trade Reward Campaign",
    "campaign_info": {
      "campaign_duration_seconds": 30000,
      "quote_denoms": [
        "quoteDenom1",
        "quoteDenom2"
      ],
      "trading_reward_boost_info": {
        "spot_market_multipliers": null,
        "boosted_derivative_market_ids": [
          "marketID"
        ],
        "derivative_market_multipliers": [
          {
            "maker_points_multiplier": "1.000000000000000000",
            "taker_points_multiplier": "3.000000000000000000"
          }
        ]
      },
      "disqualified_market_ids": [
        "marketId1",
        "marketId2"
      ]
    }
  }
}
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			proposal, err := parseBatchExchangeModificationsProposalFlags(cmd.Flags())
			if err != nil {
				return err
			}

			batchModificationMessage := types.MsgBatchExchangeModification{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&batchModificationMessage,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := v1.NewMsgSubmitProposal(
				messages,
				amount,
				clientCtx.GetFromAddress().String(),
				"",
				proposal.Title,
				proposal.Description,
				expedited,
			)
			if err != nil {
				return fmt.Errorf("invalid message: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagDeposit, "", "The proposal deposit")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewDerivativeMarketParamUpdateProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-derivative-market-params [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to update derivative market params",
		Long: `Submit a proposal to update derivative market params.

		Example:
		$ %s tx exchange update-derivative-market-params \
			--admin="inj1k2z3chspuk9wsufle69svmtmnlc07rvw9djya7" \
			--admin-permissions=1 \
			--market-id="0x000001" \
			--ticker="BTC/USDT PERP" \
			--oracle-base="BTC" \
			--oracle-quote="USDT" \
			--oracle-type="BandIBC" \
			--oracle-scale-factor="0" \
			--min-price-tick-size=4 \
			--min-quantity-tick-size=4 \
			--min-notional=1000 \
			--initial-margin-ratio="0.01" \
			--maintenance-margin-ratio="0.01" \
			--maker-fee-rate="0.01" \
			--taker-fee-rate="0.01" \
			--relayer-fee-share-rate="0.01" \
			--hourly-interest-rate="0.01" \
			--hourly-funding-rate-cap="0.00625" \
			--market-status="Active" \
			--title="INJ derivative market params update" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			marketID, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			initialMarginRatio, err := decimalFromFlag(cmd, FlagInitialMarginRatio)
			if err != nil {
				return err
			}

			maintenanceMarginRatio, err := decimalFromFlag(cmd, FlagMaintenanceMarginRatio)
			if err != nil {
				return err
			}

			makerFeeRate, err := decimalFromFlag(cmd, FlagMakerFeeRate)
			if err != nil {
				return err
			}

			takerFeeRate, err := decimalFromFlag(cmd, FlagTakerFeeRate)
			if err != nil {
				return err
			}

			relayerFeeShareRate, err := decimalFromFlag(cmd, FlagRelayerFeeShareRate)
			if err != nil {
				return err
			}

			hourlyInterestRate, err := optionalDecimalFromFlag(cmd, FlagHourlyInterestRate)
			if err != nil {
				return err
			}

			hourlyFundingRateCap, err := optionalDecimalFromFlag(cmd, FlagHourlyFundingRateCap)
			if err != nil {
				return err
			}

			minPriceTickSizeStr, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}
			minQuantityTickSizeStr, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}
			minNotionalStr, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			minPriceTickSize, err := math.LegacyNewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}
			minQuantityTickSize, err := math.LegacyNewDecFromStr(minQuantityTickSizeStr)
			if err != nil {
				return err
			}
			minNotional, err := math.LegacyNewDecFromStr(minNotionalStr)
			if err != nil {
				return err
			}

			oracleBase, err := cmd.Flags().GetString(FlagOracleBase)
			if err != nil {
				return err
			}
			oracleQuote, err := cmd.Flags().GetString(FlagOracleQuote)
			if err != nil {
				return err
			}
			oracleTypeStr, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeStr)
			if err != nil {
				return err
			}

			oracleScaleFactor, err := cmd.Flags().GetUint32(FlagOracleScaleFactor)
			if err != nil {
				return err
			}

			oracleParams := &types.OracleParams{
				OracleBase:        oracleBase,
				OracleQuote:       oracleQuote,
				OracleType:        oracleType,
				OracleScaleFactor: oracleScaleFactor,
			}

			var marketStatus *types.MarketStatus
			var status types.MarketStatus

			marketStatusStr, _ := cmd.Flags().GetString(FlagMarketStatus)
			if marketStatusStr != "" {
				m := types.MarketStatus(types.MarketStatus_value[marketStatusStr])
				marketStatus = &m
			}

			if marketStatus == nil {
				status = types.MarketStatus_Unspecified
			} else {
				status = *marketStatus
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			admin, err := cmd.Flags().GetString(FlagAdmin)
			if err != nil {
				return err
			}
			adminPermissions, err := cmd.Flags().GetUint32(FlagAdminPermissions)
			if err != nil {
				return err
			}

			content, err := derivativeMarketParamUpdateArgsToContent(
				cmd,
				marketID,
				&initialMarginRatio,
				&maintenanceMarginRatio,
				&makerFeeRate,
				&takerFeeRate,
				&relayerFeeShareRate,
				&minPriceTickSize,
				&minQuantityTickSize,
				&minNotional,
				hourlyInterestRate,
				hourlyFundingRateCap,
				oracleParams,
				status,
				ticker,
				admin,
				adminPermissions,
			)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().String(FlagMarketID, "", "ID of market to update params")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagRelayerFeeShareRate, "", "relayer fee share rate")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagHourlyInterestRate, "", "hourly interest rate")
	cmd.Flags().String(FlagHourlyFundingRateCap, "", "hourly funding rate cap")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().String(FlagMarketStatus, "", "market status")
	cmd.Flags().String(FlagTicker, "", "market ticker")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSubaccountTransferTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subaccount-transfer [src_subaccount_id] [dest_subaccount_id] [amount] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit message to send coins between subaccounts.",
		Long: `Submit message to send coins between subaccounts.

		Example:
		$ %s tx exchange subaccount-transfer 0x90f8bf6a479f320ead074411a4b0e7944ea8c9c1000000000000000000000001 0x90f8bf6a479f320ead074411a4b0e7944ea8c9c1000000000000000000000002 10000inj --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			amount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgSubaccountTransfer{
				Sender:                  from.String(),
				SourceSubaccountId:      args[0],
				DestinationSubaccountId: args[1],
				Amount:                  amount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewMarketForcedSettlementTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "force-settle-market [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to force settle market. Spot requires nil settlement price",
		Long: `Submit a proposal to force settle market. Spot requires nil settlement price

		Example:
		$ %s tx exchange force-settle-market \
			--market-id="0x000001" \
			--settlement-price="10000" \
			--title="INJ derivative market params update" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			marketID, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			settlementPriceString, err := cmd.Flags().GetString(FlagSettlementPrice)
			if err != nil {
				return err
			}

			var settlementPrice *math.LegacyDec
			if settlementPriceString != "" {
				settlementPriceValue, err := decimalFromFlag(cmd, FlagSettlementPrice)
				if err != nil {
					return err
				}
				settlementPrice = &settlementPriceValue
			} else {
				settlementPrice = nil
			}

			content, err := forcedMarketSettlementArgsToContent(
				cmd,
				marketID,
				settlementPrice,
			)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().String(FlagMarketID, "", "ID of market to update params")
	cmd.Flags().String(FlagSettlementPrice, "", "settlement price")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateDenomDecimalsProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-denom-decimals [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to update denom decimals",
		Long: `Submit a proposal to update denom decimals

		Example:
		$ %s tx exchange update-denom-decimals \
			--denoms="ibc/denom-hash1,ibc/denom-hash1" \
			--decimals="18,6" \
			--title="Decimals update" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			denoms, err := cmd.Flags().GetStringSlice(FlagDenoms)
			if err != nil {
				return err
			}

			decimals, err := cmd.Flags().GetUintSlice(FlagDecimals)
			if err != nil {
				return err
			}

			content, err := updateDenomDecimalsArgsToContent(
				cmd,
				denoms,
				decimals,
			)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().StringSlice(FlagDenoms, nil, "denoms")
	cmd.Flags().UintSlice(FlagDecimals, nil, "decimals")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateSpotMarketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-spot-market [market_id] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit admin message to update a specific spot market's fields. Must have at least one flag present!",
		Long: `Submit admin message to update a specific spot market's fields. Must have at least one flag present!\
Example:
$ %s tx exchange update-spot-market 0x1e11532fc29f1bc3eb75f6fddf4997e904c780ddf155ecb58bc89bf723e1ba56 \
	--ticker "A/B" \
	--min-price-tick-size "0.1" \
	--min-quantity-tick-size "-0.2" \
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			strMinPriceTickSize, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}

			var minPriceTickSize math.LegacyDec
			if strMinPriceTickSize != "" {
				minPriceTickSize, err = math.LegacyNewDecFromStr(strMinPriceTickSize)
				if err != nil {
					return err
				}
			}

			strMinQuantityTickSize, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}

			var minQuantityTickSize math.LegacyDec
			if strMinQuantityTickSize != "" {
				minQuantityTickSize, err = math.LegacyNewDecFromStr(strMinPriceTickSize)
				if err != nil {
					return err
				}
			}

			strMinNotional, err := cmd.Flags().GetString(FlagMinNotional)
			if err != nil {
				return err
			}

			var minNotional math.LegacyDec
			if strMinNotional != "" {
				minNotional, err = math.LegacyNewDecFromStr(strMinNotional)
				if err != nil {
					return err
				}
			}

			msg := &types.MsgUpdateSpotMarket{
				Admin:                  clientCtx.GetFromAddress().String(),
				MarketId:               common.HexToHash(args[0]).String(),
				NewTicker:              ticker,
				NewMinPriceTickSize:    minPriceTickSize,
				NewMinQuantityTickSize: minQuantityTickSize,
				NewMinNotional:         minNotional,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "new market ticker")
	cmd.Flags().String(FlagMinPriceTickSize, "", "new min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "new min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "", "new min notional")

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewUpdateDerivativeMarketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-derivative-market [market_id] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit admin message to update a specific spot market's fields. Must have at least one flag present!",
		Long: `Submit admin message to update a specific spot market's fields. Must have at least one flag present!\
Example:
$ %s tx exchange update-derivative-market 0x1e11532fc29f1bc3eb75f6fddf4997e904c780ddf155ecb58bc89bf723e1ba56 \
	--ticker "A/B" \
	--min-price-tick-size "0.1" \
	--min-quantity-tick-size "-0.2" \
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			strMinPriceTickSize, err := cmd.Flags().GetString(FlagMinPriceTickSize)
			if err != nil {
				return err
			}

			var minPriceTickSize math.LegacyDec
			if strMinPriceTickSize != "" {
				minPriceTickSize, err = math.LegacyNewDecFromStr(strMinPriceTickSize)
				if err != nil {
					return err
				}
			}

			strMinQuantityTickSize, err := cmd.Flags().GetString(FlagMinQuantityTickSize)
			if err != nil {
				return err
			}

			var minQuantityTickSize math.LegacyDec
			if strMinQuantityTickSize != "" {
				minQuantityTickSize, err = math.LegacyNewDecFromStr(strMinPriceTickSize)
				if err != nil {
					return err
				}
			}

			strInitialMarginRatio, err := cmd.Flags().GetString(FlagInitialMarginRatio)
			if err != nil {
				return err
			}

			var initialMarginRatio math.LegacyDec
			if strInitialMarginRatio != "" {
				initialMarginRatio, err = math.LegacyNewDecFromStr(strInitialMarginRatio)
				if err != nil {
					return err
				}
			}

			strMaintenanceMarginRatio, err := cmd.Flags().GetString(FlagMaintenanceMarginRatio)
			if err != nil {
				return err
			}

			var maintenanceMarginRatio math.LegacyDec
			if strMaintenanceMarginRatio != "" {
				maintenanceMarginRatio, err = math.LegacyNewDecFromStr(strMaintenanceMarginRatio)
				if err != nil {
					return err
				}
			}

			msg := &types.MsgUpdateDerivativeMarket{
				Admin:                     clientCtx.GetFromAddress().String(),
				MarketId:                  common.HexToHash(args[0]).String(),
				NewTicker:                 ticker,
				NewMinPriceTickSize:       minPriceTickSize,
				NewMinQuantityTickSize:    minQuantityTickSize,
				NewInitialMarginRatio:     initialMarginRatio,
				NewMaintenanceMarginRatio: maintenanceMarginRatio,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "new market ticker")
	cmd.Flags().String(FlagMinPriceTickSize, "", "new min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "new min quantity tick size")
	cmd.Flags().String(FlagInitialMarginRatio, "", "new initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "new maintenance margin ratio")

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewDepositTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [amount] [subaccount] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit message to transfer coins from the sender's bank balance into subaccount's exchange deposits.",
		Long: `Submit message to transfer coins from the sender's bank balance into subaccount's exchange deposits.

		Example:
		$ %s tx exchange deposit 10000inj 0xc6fe5d33615a1c52c08018c47e8bc53646a0e101000000000000000000000001 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			if args[1] == "" {
				return errors.New("subaccount id cannot be empty")
			}

			msg := &types.MsgDeposit{
				Sender:       from.String(),
				SubaccountId: args[1],
				Amount:       amount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewWithdrawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw [amount] [subaccount_id] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit message to withdraw coins from the provided subaccount's deposits to the user's bank balance.",
		Long: `Submit message to withdraw coins from the provided subaccount's deposits to the user's bank balance.

		Example:
		$ %s tx exchange withdraw 10000inj 0xc6fe5d33615a1c52c08018c47e8bc53646a0e101000000000000000000000001 \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			subaccountID := args[1]

			msg := &types.MsgWithdraw{
				Sender:       from.String(),
				SubaccountId: subaccountID,
				Amount:       amount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewExternalTransferTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "external-transfer [source_subaccount_id] [dest_subaccount_id] [amount] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit message to send coins from the sender's provided subaccount to another external subaccount.",
		Long: `Submit message to send coins from the sender's provided subaccount to another external subaccount.

		Example:
		$ %s tx exchange external-transfer [from] [to] 10000inj \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			sourceSubaccountID := args[0]
			destinationSubaccountID := args[1]

			amount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgExternalTransfer{
				Sender:                  from.String(),
				SourceSubaccountId:      sourceSubaccountID,
				DestinationSubaccountId: destinationSubaccountID,
				Amount:                  amount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewAuthzTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authz [grantee] [subaccount-id] [msg-type] [market-ids]",
		Args:  cobra.ExactArgs(4),
		Short: "Authorize grantee to execute allowed msgs in allowed markets via allowed subaccount",
		Long: `Authorize grantee to execute allowed msgs in allowed markets via allowed subaccount.

		Example:
		$ %s tx exchange authz inj1jcltmuhplrdcwp7stlr4hlhlhgd4htqhe4c0cs 0xc6fe5d33615a1c52c08018c47e8bc53646a0e101000000000000000000000000 MsgCreateSpotLimitOrder 0xa508cb32923323679f29a032c70342c147c17d0145625922b0ef22e955c844c0  --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// parse args
			grantee, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			subAccountId := args[1]
			msgType := args[2]
			marketIDs := strings.Split(args[3], ",")

			// parse optional expiration flag
			expiration, err := cmd.Flags().GetString(authzcli.FlagExpiration)
			if err != nil {
				return err
			}
			var expirationDate time.Time
			if expiration != "" {
				timestamp, err := strconv.ParseInt(expiration, 10, 64)
				if err != nil {
					panic(err)
				}
				expirationDate = time.Unix(timestamp, 0)
			} else {
				expirationDate = time.Now().AddDate(1, 0, 0)
			}

			// build msg
			authorization := buildExchangeAuthz(subAccountId, marketIDs, msgType)
			msg, err := authz.NewMsgGrant(
				clientCtx.GetFromAddress(),
				grantee,
				authorization,
				&expirationDate,
			)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(authzcli.FlagExpiration, "", "authz expiration timestamp")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewBatchUpdateAuthzTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authz batch-update [grantee] [subaccount-id] [spot-market-ids] [derivative-market-ids]",
		Args:  cobra.ExactArgs(4),
		Short: "Authorize grantee to execute MsgBatchUpdateOrders in allowed markets via allowed subaccount",
		Long: `Authorize grantee to execute MsgBatchUpdateOrders in allowed markets via allowed subaccount.

		Example:
		$ %s tx exchange authz batch-update inj1jcltmuhplrdcwp7stlr4hlhlhgd4htqhe4c0cs 0xc6fe5d33615a1c52c08018c47e8bc53646a0e101000000000000000000000000 0xa508cb32923323679f29a032c70342c147c17d0145625922b0ef22e955c844c0 0xfd30930cb70d176c37d0c405cde055e551c5b1116b7049a88bcf821766b62d62 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// parse args
			grantee, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			subAccountId := args[1]
			spotMarketIDs := strings.Split(args[2], ",")
			derivativeMarketIDs := strings.Split(args[3], ",")

			// parse optional expiration flag
			expiration, err := cmd.Flags().GetString(authzcli.FlagExpiration)
			if err != nil {
				return err
			}
			var expirationDate time.Time
			if expiration != "" {
				timestamp, err := strconv.ParseInt(expiration, 10, 64)
				if err != nil {
					panic(err)
				}
				expirationDate = time.Unix(timestamp, 0)
			} else {
				expirationDate = time.Now().AddDate(1, 0, 0)
			}

			// build msg
			authorization := buildBatchUpdateExchangeAuthz(
				subAccountId,
				spotMarketIDs,
				derivativeMarketIDs,
			)
			msg, err := authz.NewMsgGrant(
				clientCtx.GetFromAddress(),
				grantee,
				authorization,
				&expirationDate,
			)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(authzcli.FlagExpiration, "", "authz expiration timestamp")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRewardsOptOutTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewards-opt-out",
		Args:  cobra.ExactArgs(1),
		Short: "Register this user to opt out of trading rewards.",
		Long: `Register this user to opt out of trading rewards

		Example:
		$ %s tx exchange rewards-opt-out --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// build msg
			msg := &types.MsgRewardsOptOut{
				Sender: clientCtx.GetFromAddress().String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewAtomicMarketOrderFeeMultiplierScheduleProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-atomic-fee-multiplier [marketId:multiplier] [flags]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Submit a proposal to set atomic market order fee multiplier for given markets",
		Long: `Submit a proposal to set atomic market order fee multiplier for given markets.

		Example:
		$ %s tx exchange propose-atomic-fee-multiplier 0xfd30930cb70d176c37d0c405cde055e551c5b1116b7049a88bcf821766b62d61:3.0 0xfd30930cb70d176c37d0c405cde055e551c5b1116b7049a88bcf821766b62d62:2.0  \
			--title="Set Atomic Orders Fees Multiplier" \
			--description="Set Atomic Orders Fees Multiplier" \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			multipliers := make([]*types.MarketFeeMultiplier, 0)
			for _, arg := range args {
				split := strings.Split(arg, ":")
				if len(split) != 2 {
					return types.ErrInvalidArgument.Wrapf(
						"%v does not match a pattern marketId:multiplier",
						arg,
					)
				}
				marketId := split[0]
				common.HexToHash(marketId)
				feeMultiplier, err := math.LegacyNewDecFromStr(split[1])
				if err != nil {
					return err
				}
				multiplier := types.MarketFeeMultiplier{
					MarketId:      marketId,
					FeeMultiplier: feeMultiplier,
				}
				multipliers = append(multipliers, &multiplier)
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			content := &types.AtomicMarketOrderFeeMultiplierScheduleProposal{
				Title:                title,
				Description:          description,
				MarketFeeMultipliers: multipliers,
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewStakeGrantAuthorizationTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authorize-stake-grant <stake_grants.json>",
		Args:  cobra.ExactArgs(1),
		Short: "Authorize grantee a given amount of staked INJ tokens for fee tier discounts",
		Long: `Authorize grantee a given amount of staked INJ tokens for fee tier discounts.

		Example:
		$ %s tx exchange authorize-stake-grant stake_grants.json \ 
			
			Where stake_grant_authorizations.json contains:
			{
				"grants":
				[
					{
						"grantee": "inj1jcltmuhplrdcwp7stlr4hlhlhgd4htqhe4c0cs",
						"amount": "1000000000"
					},
					{
						"grantee": "inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c",
						"amount": "321000"
					},
					{   
						"grantee": "inj1hdvy6tl89llqy3ze8lv6mz5qh66sx9enn0jxg6",
						"amount": "20"
					}
				]
			}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			file, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}

			msg := &types.MsgAuthorizeStakeGrants{}
			if err := json.Unmarshal(file, &msg); err != nil {
				return err
			}
			msg.Sender = clientCtx.GetFromAddress().String()

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewStakeGrantActivationTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate-stake-grant [granter]",
		Args:  cobra.ExactArgs(1),
		Short: "Activate a stake grant previously authorized by the granter",
		Long: `Activate a stake grant previously authorized by the granter. \

		Example:
		$ %s tx exchange activate-stake-grant inj1jcltmuhplrdcwp7stlr4hlhlhgd4htqhe4c0cs 
			--yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee := clientCtx.GetFromAddress()
			granter := args[0]
			msg := types.MsgActivateStakeGrant{
				Sender:  grantee.String(),
				Granter: granter,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewIncreasePositionMarginTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"increase-position-margin <source-subaccount-id> <dest-subaccount-id> <market-id> <amount>",
		"Increase margin in an open position",
		&types.MsgIncreasePositionMargin{},
		nil,
		cli.ArgsMapping{
			"SourceSubaccountId":      cli.Arg{Index: 0},
			"DestinationSubaccountId": cli.Arg{Index: 1},
			"MarketId":                cli.Arg{Index: 2, Transform: getDerivativeMarketIdFromTicker},
			"Amount":                  cli.Arg{Index: 3},
		},
	)
	cmd.Example = `injectived tx exchange increase-position-margin 0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 "ETH/USDT PERP" 10000000`
	return cmd
}

func NewDecreasePositionMarginTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"decrease-position-margin <source-subaccount-id> <dest-subaccount-id> <market-id> <amount>",
		"Decrease margin in an open position",
		&types.MsgDecreasePositionMargin{},
		nil,
		cli.ArgsMapping{
			"SourceSubaccountId":      cli.Arg{Index: 0},
			"DestinationSubaccountId": cli.Arg{Index: 1},
			"MarketId":                cli.Arg{Index: 2, Transform: getDerivativeMarketIdFromTicker},
			"Amount":                  cli.Arg{Index: 3},
		},
	)
	cmd.Example = `injectived tx exchange decrease-position-margin 0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 "ETH/USDT PERP" 10000000`
	return cmd
}

func NewMsgLiquidatePositionTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidate-position <subaccount-id> <market-id>",
		Args:  cobra.ExactArgs(2),
		Short: "Liquidate a position",
		Long: `Liquidate a position

		Example:
		$ %s tx exchange liquidate-position 0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 0x77261d2236f465ca70995043e4134897bcf8aee1262ba69d93ad819d5722cd6a
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// build msg
			msg := &types.MsgLiquidatePosition{
				Sender:       clientCtx.GetFromAddress().String(),
				SubaccountId: args[0],
				MarketId:     args[1],
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getSpotMarketIdFromTicker(ticker string, ctx grpc.ClientConn) (any, error) {
	queryClient := types.NewQueryClient(ctx)
	req := &types.QuerySpotMarketsRequest{
		Status: "Active",
	}
	res, err := queryClient.SpotMarkets(context.Background(), req)
	if err != nil {
		return nil, err
	}
	var market *types.SpotMarket
	for _, spotMarket := range res.Markets {
		if spotMarket.Ticker == ticker {
			market = spotMarket
		}
	}
	if market == nil { // not a ticker? but marketId itself?
		return ticker, nil
	}
	return market.MarketId, nil
}

func getDerivativeMarketIdFromTicker(ticker string, ctx grpc.ClientConn) (any, error) {
	queryClient := types.NewQueryClient(ctx)
	req := &types.QueryDerivativeMarketsRequest{}
	res, err := queryClient.DerivativeMarkets(context.Background(), req)
	if err != nil {
		return nil, err
	}
	var market *types.DerivativeMarket
	for _, derivativeMarket := range res.Markets {
		if derivativeMarket.Market.Ticker == ticker {
			market = derivativeMarket.Market
		}
	}
	if market == nil {
		return ticker, nil
	}
	return market.MarketId, nil
}

func expiryFuturesMarketLaunchArgsToContent(
	cmd *cobra.Command,
	ticker, quoteDenom, oracleBase, oracleQuote string,
	oracleScaleFactor uint32,
	oracleType oracletypes.OracleType,
	expiry int64,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := types.NewExpiryFuturesMarketLaunchProposal(
		title,
		description,
		ticker,
		quoteDenom,
		oracleBase,
		oracleQuote,
		oracleScaleFactor,
		oracleType,
		expiry,
		initialMarginRatio,
		maintenanceMarginRatio,
		makerFeeRate,
		takerFeeRate,
		minPriceTickSize,
		minQuantityTickSize,
		minNotional,
	)
	return content, nil
}

func derivativeMarketParamUpdateArgsToContent(
	cmd *cobra.Command,
	marketID string,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize, minNotional *math.LegacyDec,
	hourlyInterestRate, hourlyFundingRateCap *math.LegacyDec,
	oracleParams *types.OracleParams,
	status types.MarketStatus,
	ticker string,
	admin string,
	adminPermissions uint32,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	var adminInfo *types.AdminInfo

	if admin != "" {
		adminInfo = &types.AdminInfo{
			Admin:            admin,
			AdminPermissions: adminPermissions,
		}
	}

	content := types.NewDerivativeMarketParamUpdateProposal(
		title,
		description,
		marketID,
		initialMarginRatio,
		maintenanceMarginRatio,
		makerFeeRate,
		takerFeeRate,
		relayerFeeShareRate,
		minPriceTickSize,
		minQuantityTickSize,
		minNotional,
		hourlyInterestRate,
		hourlyFundingRateCap,
		status,
		oracleParams,
		ticker,
		adminInfo,
	)
	return content, nil
}

func forcedMarketSettlementArgsToContent(
	cmd *cobra.Command, marketID string,
	settlementPrice *math.LegacyDec,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := types.NewMarketForcedSettlementProposal(
		title,
		description,
		marketID,
		settlementPrice,
	)
	return content, nil
}

func updateDenomDecimalsArgsToContent(
	cmd *cobra.Command, denoms []string,
	decimals []uint,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	denomDecimals := make([]*types.DenomDecimals, 0, len(denoms))
	for idx, denom := range denoms {
		denomDecimals = append(denomDecimals, &types.DenomDecimals{
			Denom:    denom,
			Decimals: uint64(decimals[idx]),
		})
	}

	content := types.NewUpdateDenomDecimalsProposal(
		title,
		description,
		denomDecimals,
	)
	return content, nil
}

func decimalFromFlag(cmd *cobra.Command, flag string) (math.LegacyDec, error) {
	decStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return math.LegacyNewDecFromStr(decStr)
}

func optionalDecimalFromFlag(cmd *cobra.Command, flag string) (*math.LegacyDec, error) {
	decStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}

	if decStr == "" {
		return nil, nil
	}

	valueDec, err := math.LegacyNewDecFromStr(decStr)
	return &valueDec, err
}

func buildExchangeAuthz(
	subaccountId string,
	marketIDs []string,
	msgType string,
) authz.Authorization {
	switch msgType {
	// spot messages
	case "MsgCreateSpotLimitOrder":
		return &types.CreateSpotLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCreateSpotMarketOrder":
		return &types.CreateSpotMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCreateSpotLimitOrders":
		return &types.BatchCreateSpotLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCancelSpotOrder":
		return &types.CancelSpotOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCancelSpotOrders":
		return &types.BatchCancelSpotOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}

	// derivative messages
	case "MsgCreateDerivativeLimitOrder":
		return &types.CreateDerivativeLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCreateDerivativeMarketOrder":
		return &types.CreateDerivativeMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCreateDerivativeLimitOrders":
		return &types.BatchCreateDerivativeLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCancelDerivativeOrder":
		return &types.CancelDerivativeOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCancelDerivativeOrders":
		return &types.BatchCancelDerivativeOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	default:
		panic("Invalid or unsupported exchange message type to authorize")
	}
}

func buildBatchUpdateExchangeAuthz(
	subaccountId string,
	spotMarketIDs, derivativeMarketIDs []string,
) authz.Authorization {
	return &types.BatchUpdateOrdersAuthz{
		SubaccountId:      subaccountId,
		SpotMarkets:       spotMarketIDs,
		DerivativeMarkets: derivativeMarketIDs,
	}
}

func defineDerivativeOrderFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagMarketID, "", "market id")
	cmd.Flags().String(FlagSubaccountID, "", "subaccount id")
	cmd.Flags().String(FlagFeeRecipient, "", "fee recipient")
	cmd.Flags().String(FlagPrice, "", "price")
	cmd.Flags().String(FlagQuantity, "", "quantity")
	cmd.Flags().String(FlagOrderType, "", "order type")
	cmd.Flags().String(FlagTriggerPrice, "", "trigger price")
	cmd.Flags().Bool(FlagReduceOnly, false, "reduce only")
	cmd.Flags().String(FlagCID, "", "client order id")
	cliflags.AddTxFlagsToCmd(cmd)
}
