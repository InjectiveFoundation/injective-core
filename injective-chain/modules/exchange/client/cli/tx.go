//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzcli "github.com/cosmos/cosmos-sdk/x/authz/client/cli"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
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
		// other
		NewExchangeEnableProposalTxCmd(),
		NewMarketForcedSettlementTxCmd(),
		NewUpdateDenomDecimalsProposalTxCmd(),
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
		},
		cli.ArgsMapping{},
	)
	cmd.Example = `tx exchange instant-spot-market-launch INJ/ATOM uinj uatom --min-price-tick-size=1000000000 --min-quantity-tick-size=1000000000000000`
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
	return cmd
}

func NewCreateSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-limit-order <order_type> <market_ticker> <quantity> <price>",
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
		},
	)
	cmd.Example = "injectived tx exchange create-spot-limit-order buy ETH/USDT 2.4 2000.1 --from=genesis --keyring-backend=file --yes"
	return cmd
}

func NewCreateSpotMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-market-order <order_type> <market_ticker> <quantity> <worst_price>",
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
		},
	)
	cmd.Example = "injectived tx exchange create-spot-limit-order buy ETH/USDT 2.4 2000.1 --from=genesis --keyring-backend=file --yes"
	return cmd
}

func NewCancelSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-spot-limit-order <market_ticker> <order_hash>",
		"Cancel Spot Limit Order",
		&types.MsgCancelSpotOrder{},
		cli.FlagsMapping{},
		cli.ArgsMapping{"MarketId": cli.Arg{Index: 0, Transform: getSpotMarketIdFromTicker}},
	)
	cmd.Example = "injectived tx exchange cancel-spot-limit-order ETH/USDT 0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8"
	return cmd
}

func NewCancelDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-derivative-limit-order <market_ticker> <order_hash>",
		"Cancel Derivative Limit Order",
		&types.MsgCancelDerivativeOrder{},
		cli.FlagsMapping{},
		cli.ArgsMapping{"MarketId": cli.Arg{Index: 0, Transform: getDerivativeMarketIdFromTicker}},
	)
	cmd.Example = "tx exchange cancel-derivative-limit-order ETH/USDT 0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8"
	return cmd
}

func NewCreateDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-derivative-limit-order",
		"Create Derivative Limit Order",
		&types.MsgCreateDerivativeLimitOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.SkipField, // disable parsing of trigger price
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
					default:
						return orderType, fmt.Errorf(
							`order type must be "buy", "sell", "buy-PO" or "sell-PO"`,
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
			--from=genesis \
			--keyring-backend=file \
			--yes`
	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	return cmd
}

func NewCreateDerivativeMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-derivative-market-order",
		"Create Derivative Market Order",
		&types.MsgCreateDerivativeMarketOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.SkipField, // disable parsing of trigger price
			"OrderType": cli.Flag{
				Flag: FlagOrderType,
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
			"MarketId": cli.Flag{
				Flag:      FlagMarketID,
				Transform: getDerivativeMarketIdFromTicker,
			},
			"Price":        cli.Flag{Flag: FlagPrice},
			"Quantity":     cli.Flag{Flag: FlagQuantity},
			"Margin":       cli.Flag{Flag: FlagMargin},
			"SubaccountId": cli.Flag{Flag: FlagSubaccountID},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = `tx exchange create-derivative-market-order \
			--order-type="buy" \
			--market-id="ETH/USDT" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--price="4.1" \
			--quantity="10.01" \
			--margin="30.0"`
	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	return cmd
}

func NewSpotMarketUpdateParamsProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.SpotMarketParamUpdateProposal{})
	cmd := cli.TxCmd(
		"update-spot-market-params",
		"Submit a proposal to update spot market params",
		proposalMsgDummy, cli.FlagsMapping{
			"Title":               cli.Flag{Flag: govcli.FlagTitle},
			"Description":         cli.Flag{Flag: govcli.FlagDescription},
			"MarketId":            cli.Flag{Flag: FlagMarketID},
			"MakerFeeRate":        cli.Flag{Flag: FlagMakerFeeRate},
			"TakerFeeRate":        cli.Flag{Flag: FlagTakerFeeRate},
			"RelayerFeeShareRate": cli.Flag{Flag: FlagRelayerFeeShareRate},
			"MinPriceTickSize":    cli.Flag{Flag: FlagMinPriceTickSize},
			"MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize},
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
			"InitialDeposit": cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange update-spot-market-params --market-id="0xacdd4f9cb90ecf5c4e254acbf65a942f562ca33ba718737a93e5cb3caadec3aa" --title="Spot market params update" --description="XX" --deposit="1000000000000000000inj"`

	cmd.Flags().String(FlagMarketID, "", "Spot market ID")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagRelayerFeeShareRate, "", "relayer fee share rate")
	cmd.Flags().String(FlagMinPriceTickSize, "", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "min quantity tick size")
	cmd.Flags().String(FlagMarketStatus, "", "market status")
	cliflags.AddGovProposalFlags(cmd)

	return cmd
}

func NewSpotMarketLaunchProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.SpotMarketLaunchProposal{})
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
			"InitialDeposit":      cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=1000000000 \
			--min-quantity-tick-size=1000000000000000 \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--title="INJ/ATOM spot market" \
			--description="XX" \
			--deposit="1000000000000000000inj"`

	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
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
	_ = proposalMsgDummy.SetContent(&types.PerpetualMarketLaunchProposal{})
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

			minPriceTickSize, err := sdk.NewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
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

			minPriceTickSize, err := sdk.NewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
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

			minPriceTickSize, err := sdk.NewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
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

			var settlementPrice *sdk.Dec
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
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			marketId, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			subaccountId, err := cmd.Flags().GetString(FlagSubaccountID)
			if err != nil {
				return err
			}

			txHash, err := cmd.Flags().GetString(FlagOrderHash)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelBinaryOptionsOrder{
				Sender:       clientCtx.GetFromAddress().String(),
				MarketId:     marketId,
				SubaccountId: subaccountId,
				OrderHash:    txHash,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketID, "", "market id")
	cmd.Flags().String(FlagSubaccountID, "", "subaccount id")
	cmd.Flags().String(FlagOrderHash, "", "order (trasnasction) hash")
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

			minPriceTickSize, err := sdk.NewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
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
$ %s tx gov batch-exchange-modifications-proposal --proposal="path/to/proposal.json" --from=mykey --deposit=1000000inj

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

			content, err := parseBatchExchangeModificationsProposalFlags(cmd.Flags())
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, amount, clientCtx.GetFromAddress())
			if err != nil {
				return fmt.Errorf("invalid message: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagDeposit, "", "The proposal deposit")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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
			--market-id="0x000001" \
			--oracle-base="BTC" \
			--oracle-quote="USDT" \
			--oracle-type="BandIBC" \
			--oracle-scale-factor="0" \
			--min-price-tick-size=4 \
			--min-quantity-tick-size=4 \
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

			minPriceTickSize, err := sdk.NewDecFromStr(minPriceTickSizeStr)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
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
				hourlyInterestRate,
				hourlyFundingRateCap,
				oracleParams,
				status,
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
	cmd.Flags().String(FlagHourlyInterestRate, "", "hourly interest rate")
	cmd.Flags().String(FlagHourlyFundingRateCap, "", "hourly funding rate cap")
	cmd.Flags().String(FlagOracleBase, "", "oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "oracle quote")
	cmd.Flags().String(FlagOracleType, "", "oracle type")
	cmd.Flags().Uint32(FlagOracleScaleFactor, 0, "oracle scale factor")
	cmd.Flags().String(FlagMarketStatus, "", "market status")

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

			var settlementPrice *sdk.Dec
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
		Use:   "withdraw [amount] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit message to withdraw coins from the default subaccount's deposits to the user's bank balance.",
		Long: `Submit message to withdraw coins from the default subaccount's deposits to the user's bank balance.

		Example:
		$ %s tx exchange withdraw 10000inj --from=genesis --keyring-backend=file --yes
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

			subaccountID := types.SdkAddressToSubaccountID(from)

			msg := &types.MsgWithdraw{
				Sender:       from.String(),
				SubaccountId: subaccountID.Hex(),
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
		Use:   "external-transfer [dest_subaccount_id] [amount] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit message to send coins from the sender's default subaccount to another external subaccount.",
		Long: `Submit message to send coins from the sender's default subaccount to another external subaccount",

		Example:
		$ %s tx exchange external-transfer 0x90f8bf6a479f320ead074411a4b0e7944ea8c9c1000000000000000000000001 10000inj --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			amount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			subaccountID := types.SdkAddressToSubaccountID(from)

			msg := &types.MsgExternalTransfer{
				Sender:                  from.String(),
				SourceSubaccountId:      subaccountID.Hex(),
				DestinationSubaccountId: args[0],
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
			marketIds := strings.Split(args[3], ",")

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
			authorization := buildExchangeAuthz(subAccountId, marketIds, msgType)
			msg, err := authz.NewMsgGrant(
				clientCtx.GetFromAddress(),
				grantee,
				authorization,
				&expirationDate,
			)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
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
			spotMarketIds := strings.Split(args[2], ",")
			derivativeMarketIds := strings.Split(args[3], ",")

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
				spotMarketIds,
				derivativeMarketIds,
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

			if err := msg.ValidateBasic(); err != nil {
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
				feeMultiplier, err := sdk.NewDecFromStr(split[1])
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
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize sdk.Dec,
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
	)
	return content, nil
}

func derivativeMarketParamUpdateArgsToContent(
	cmd *cobra.Command,
	marketID string,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize *sdk.Dec,
	hourlyInterestRate, hourlyFundingRateCap *sdk.Dec,
	oracleParams *types.OracleParams,
	status types.MarketStatus,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
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
		hourlyInterestRate,
		hourlyFundingRateCap,
		status,
		oracleParams,
	)
	return content, nil
}

func forcedMarketSettlementArgsToContent(
	cmd *cobra.Command, marketID string,
	settlementPrice *sdk.Dec,
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

func decimalFromFlag(cmd *cobra.Command, flag string) (sdk.Dec, error) {
	decStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return sdk.Dec{}, err
	}

	return sdk.NewDecFromStr(decStr)
}

func optionalDecimalFromFlag(cmd *cobra.Command, flag string) (*sdk.Dec, error) {
	decStr, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil, err
	}

	if decStr == "" {
		return nil, nil
	}

	valueDec, err := sdk.NewDecFromStr(decStr)
	return &valueDec, err
}

func buildExchangeAuthz(
	subaccountId string,
	marketIds []string,
	msgType string,
) authz.Authorization {
	switch msgType {
	// spot messages
	case "MsgCreateSpotLimitOrder":
		return &types.CreateSpotLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgCreateSpotMarketOrder":
		return &types.CreateSpotMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgBatchCreateSpotLimitOrders":
		return &types.BatchCreateSpotLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgCancelSpotOrder":
		return &types.CancelSpotOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgBatchCancelSpotOrders":
		return &types.BatchCancelSpotOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}

	// derivative messages
	case "MsgCreateDerivativeLimitOrder":
		return &types.CreateDerivativeLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgCreateDerivativeMarketOrder":
		return &types.CreateDerivativeMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgBatchCreateDerivativeLimitOrders":
		return &types.BatchCreateDerivativeLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgCancelDerivativeOrder":
		return &types.CancelDerivativeOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	case "MsgBatchCancelDerivativeOrders":
		return &types.BatchCancelDerivativeOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIds,
		}
	default:
		panic("Invalid or unsupported exchange message type to authorize")
	}
}

func buildBatchUpdateExchangeAuthz(
	subaccountId string,
	spotMarketIds, derivativeMarketIds []string,
) authz.Authorization {
	return &types.BatchUpdateOrdersAuthz{
		SubaccountId:      subaccountId,
		SpotMarkets:       spotMarketIds,
		DerivativeMarkets: derivativeMarketIds,
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
	cliflags.AddTxFlagsToCmd(cmd)
}
