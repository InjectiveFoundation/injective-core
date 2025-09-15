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

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcli "github.com/cosmos/cosmos-sdk/x/authz/client/cli"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govgeneraltypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
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
		NewSetDelegationTransferReceiversTxCmd(),
		// other
		NewExchangeEnableProposalTxCmd(),
		NewMarketForcedSettlementTxCmd(),
		NewUpdateDenomDecimalsProposalTxCmd(),
		NewIncreasePositionMarginTxCmd(),
		NewDecreasePositionMarginTxCmd(),
		NewMsgLiquidatePositionTxCmd(),
		NewCancelPostOnlyModeTxCmd(),
	)
	return cmd
}

func NewInstantSpotMarketLaunchTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"instant-spot-market-launch <ticker> <base_denom> <quote_denom>",
		"Launch spot market by paying listing fee without governance",
		&exchangev2.MsgInstantSpotMarketLaunch{},
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
			--min-price-tick-size=0.01 \
			--min-quantity-tick-size=0.001 \
			--min-notional=1 \
			--base-decimals=18 \
			--quote-decimals=6`
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.001", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagBaseDecimals, "0", "base token decimals")
	cmd.Flags().String(FlagQuoteDecimals, "0", "quote token decimals")
	return cmd
}

func NewCreateSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-limit-order <order_type> <market_ticker> <quantity> <price> <client_order_id>",
		"Create Spot Limit Order",
		&exchangev2.MsgCreateSpotLimitOrder{},
		cli.FlagsMapping{
			"ExpirationBlock": cli.Flag{Flag: FlagExpirationBlock, UseDefaultIfOmitted: true},
			"TriggerPrice":    cli.SkipField, // disable parsing of trigger price
		},
		cli.ArgsMapping{
			"OrderType": cli.Arg{
				Index: 0,
				Transform: func(orig string, _ grpc.ClientConn) (any, error) {
					var orderType exchangev2.OrderType
					switch orig {
					case "buy":
						orderType = exchangev2.OrderType_BUY
					case "sell":
						orderType = exchangev2.OrderType_SELL
					case "buy-PO":
						orderType = exchangev2.OrderType_BUY_PO
					case "sell-PO":
						orderType = exchangev2.OrderType_SELL_PO
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
	cmd.Flags().String(FlagExpirationBlock, "0", "expiration block")
	return cmd
}

func NewCreateSpotMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-spot-market-order <order_type> <market_ticker> <quantity> <worst_price> <client_order_id>",
		"Create Spot Market Order",
		&exchangev2.MsgCreateSpotMarketOrder{},
		cli.FlagsMapping{
			"TriggerPrice":    cli.SkipField, // disable parsing of trigger price
			"ExpirationBlock": cli.SkipField, // disable parsing of expiration block for market orders
		},
		cli.ArgsMapping{
			"OrderType": cli.Arg{
				Index: 0,
				Transform: func(orig string, _ grpc.ClientConn) (any, error) {
					var orderType exchangev2.OrderType
					switch orig {
					case "buy":
						orderType = exchangev2.OrderType_BUY
					case "sell":
						orderType = exchangev2.OrderType_SELL
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
	cmd.Example = "injectived tx exchange create-spot-market-order buy ETH/USDT 2.4 2.1 order_1 --from=genesis --keyring-backend=file --yes"
	return cmd
}

func NewCancelSpotLimitOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-spot-limit-order",
		"Cancel Spot Limit Order",
		&exchangev2.MsgCancelSpotOrder{},
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
		&exchangev2.MsgCancelDerivativeOrder{},
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
		&exchangev2.MsgCreateDerivativeLimitOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.Flag{Flag: FlagTriggerPrice},
			"OrderType": cli.Flag{
				Flag: FlagOrderType,
				Transform: func(orig string, _ grpc.ClientConn) (any, error) {
					var orderType exchangev2.OrderType
					switch orig {
					case "buy":
						orderType = exchangev2.OrderType_BUY
					case "sell":
						orderType = exchangev2.OrderType_SELL
					case "buy-PO":
						orderType = exchangev2.OrderType_BUY_PO
					case "sell-PO":
						orderType = exchangev2.OrderType_SELL_PO
					case "take-sell":
						orderType = exchangev2.OrderType_TAKE_SELL
					case "stop-sell":
						orderType = exchangev2.OrderType_STOP_SELL
					case "stop-buy":
						orderType = exchangev2.OrderType_STOP_BUY
					case "take-buy":
						orderType = exchangev2.OrderType_TAKE_BUY
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
			"Price":           cli.Flag{Flag: FlagPrice},
			"Quantity":        cli.Flag{Flag: FlagQuantity},
			"Margin":          cli.Flag{Flag: FlagMargin},
			"SubaccountId":    cli.Flag{Flag: FlagSubaccountID},
			"Cid":             cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
			"ExpirationBlock": cli.Flag{Flag: FlagExpirationBlock, UseDefaultIfOmitted: true},
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
	cmd.Flags().String(FlagExpirationBlock, "0", "Expiration block")
	return cmd
}

func NewCreateDerivativeMarketOrderTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"create-derivative-market-order",
		"Create Derivative Market Order",
		&exchangev2.MsgCreateDerivativeMarketOrder{},
		cli.FlagsMapping{
			"TriggerPrice": cli.Flag{Flag: FlagTriggerPrice},
			"OrderType": cli.Flag{
				Flag: FlagOrderType,
				Transform: func(orig string, _ grpc.ClientConn) (any, error) {
					var orderType exchangev2.OrderType
					switch orig {
					case "buy":
						orderType = exchangev2.OrderType_BUY
					case "sell":
						orderType = exchangev2.OrderType_SELL
					case "take-sell":
						orderType = exchangev2.OrderType_TAKE_SELL
					case "stop-sell":
						orderType = exchangev2.OrderType_STOP_SELL
					case "stop-buy":
						orderType = exchangev2.OrderType_STOP_BUY
					case "take-buy":
						orderType = exchangev2.OrderType_TAKE_BUY
					case "buy-atomic":
						orderType = exchangev2.OrderType_BUY_ATOMIC
					case "sell-atomic":
						orderType = exchangev2.OrderType_SELL_ATOMIC
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
			"Price":           cli.Flag{Flag: FlagPrice},
			"Quantity":        cli.Flag{Flag: FlagQuantity},
			"Margin":          cli.Flag{Flag: FlagMargin},
			"SubaccountId":    cli.Flag{Flag: FlagSubaccountID},
			"Cid":             cli.Flag{Flag: FlagCID, UseDefaultIfOmitted: true},
			"ExpirationBlock": cli.SkipField, // disable parsing of expiration block for market orders
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
	cmd := &cobra.Command{
		Use:   "update-spot-market-params",
		Short: "Submit a proposal to update spot market params",
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

			flagsMapping := cli.FlagsMapping{
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
					Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
						var status exchangev2.MarketStatus
						if origV != "" {
							newStatus, ok := exchangev2.MarketStatus_value[origV]
							if !ok {
								return nil, fmt.Errorf("incorrect market status: %s", origV)
							}
							status = exchangev2.MarketStatus(newStatus)
						} else {
							status = exchangev2.MarketStatus_Unspecified
						}
						return fmt.Sprintf("%v", int32(status)), nil
					},
				},
				"BaseDecimals":  cli.Flag{Flag: FlagBaseDecimals},
				"QuoteDecimals": cli.Flag{Flag: FlagQuoteDecimals},
			}
			argsMapping := cli.ArgsMapping{}

			proposal := exchangev2.SpotMarketParamUpdateProposal{
				AdminInfo: &exchangev2.AdminInfo{},
			}

			err = cli.ParseFieldsFromFlagsAndArgs(&proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
			if err != nil {
				return err
			}

			message := exchangev2.MsgSpotMarketParamUpdate{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: &proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
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

	cmd.Example = `tx exchange update-spot-market-params \
			--market-id="0xacdd4f9cb90ecf5c4e254acbf65a942f562ca33ba718737a93e5cb3caadec3aa" \
			--base-decimals=18 \
			--quote-decimals=6 \
			--title="Spot market params update" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--expedited=false`

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
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewSpotMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-market-launch <ticker> <base_denom> <quote_denom>",
		Short: "Submit a proposal to launch spot-market",
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

			flagsMapping := cli.FlagsMapping{
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
			}
			argsMapping := cli.ArgsMapping{}

			proposal := exchangev2.SpotMarketLaunchProposal{
				AdminInfo: &exchangev2.AdminInfo{},
			}

			err = cli.ParseFieldsFromFlagsAndArgs(&proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
			if err != nil {
				return err
			}

			message := exchangev2.MsgSpotMarketLaunch{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: &proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
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

	cmd.Example = `tx exchange spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=0.01 \
			--min-quantity-tick-size=0.001 \
			--min-notional=1 \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--base-decimals=18 \
			--quote-decimals=6 \
			--title="INJ/ATOM spot market" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--expedited=false`

	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")
	cmd.Flags().String(FlagBaseDecimals, "0", "base token decimals")
	cmd.Flags().String(FlagQuoteDecimals, "0", "quote token decimals")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewExchangeEnableProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-exchange-enable <exchange-type>",
		Short: "Submit a proposal to enable spot or derivatives exchange (exchangeType of spot or derivatives)",
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

			flagsMapping := cli.FlagsMapping{
				"Title":       cli.Flag{Flag: govcli.FlagTitle},
				"Description": cli.Flag{Flag: govcli.FlagDescription},
			}
			argsMapping := cli.ArgsMapping{
				"ExchangeType": cli.Arg{
					Index: 0,
					Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
						var exchangeType exchangev2.ExchangeType
						switch origV {
						case "spot":
							exchangeType = exchangev2.ExchangeType_SPOT
						case "derivatives":
							exchangeType = exchangev2.ExchangeType_DERIVATIVES
						default:
							return nil, fmt.Errorf("incorrect exchange type %s", origV)
						}
						return fmt.Sprintf("%v", int32(exchangeType)), nil
					},
				},
			}

			proposal := exchangev2.ExchangeEnableProposal{}

			err = cli.ParseFieldsFromFlagsAndArgs(&proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
			if err != nil {
				return err
			}

			message := exchangev2.MsgExchangeEnable{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: &proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
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

	cmd.Example = `tx exchange spot --title="Enable Spot Exchange" --description="Enable Spot Exchange" --deposit="1000000000000000000inj" --expedited=false`

	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewPerpetualMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-perpetual-market",
		Short: "Submit a proposal to launch perpetual market",
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

			flagsMapping := cli.FlagsMapping{
				"Title":             cli.Flag{Flag: govcli.FlagTitle},
				"Description":       cli.Flag{Flag: govcli.FlagDescription},
				"Ticker":            cli.Flag{Flag: FlagTicker},
				"QuoteDenom":        cli.Flag{Flag: FlagQuoteDenom},
				"OracleBase":        cli.Flag{Flag: FlagOracleBase},
				"OracleQuote":       cli.Flag{Flag: FlagOracleQuote},
				"OracleScaleFactor": cli.Flag{Flag: FlagOracleScaleFactor},
				"OracleType": cli.Flag{
					Flag: FlagOracleType,
					Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
						oracleType, err := oracletypes.GetOracleType(origV)
						if err != nil {
							return nil, fmt.Errorf("error parsing oracle type: %w", err)
						}
						return fmt.Sprintf("%v", int32(oracleType)), nil
					},
				},
				"InitialMarginRatio":     cli.Flag{Flag: FlagInitialMarginRatio},
				"MaintenanceMarginRatio": cli.Flag{Flag: FlagMaintenanceMarginRatio},
				"ReduceMarginRatio":      cli.Flag{Flag: FlagReduceMarginRatio},
				"MakerFeeRate":           cli.Flag{Flag: FlagMakerFeeRate},
				"TakerFeeRate":           cli.Flag{Flag: FlagTakerFeeRate},
				"MinPriceTickSize":       cli.Flag{Flag: FlagMinPriceTickSize},
				"MinQuantityTickSize":    cli.Flag{Flag: FlagMinQuantityTickSize},
				"MinNotional":            cli.Flag{Flag: FlagMinNotional},
				"Admin":                  cli.Flag{Flag: FlagAdmin},
				"AdminPermissions":       cli.Flag{Flag: FlagAdminPermissions},
			}
			argsMapping := cli.ArgsMapping{}

			proposal := exchangev2.PerpetualMarketLaunchProposal{
				AdminInfo: &exchangev2.AdminInfo{},
			}

			err = cli.ParseFieldsFromFlagsAndArgs(&proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
			if err != nil {
				return err
			}

			message := exchangev2.MsgPerpetualMarketLaunch{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: &proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
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
			--min-price-tick-size="0.01" \
			--min-quantity-tick-size="0.001" \
			--min-notional="1" \
			--title="INJ perpetual market" \
			--description="XX" \
			--deposit="1000000000000000000inj" \
			--expedited=false`
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
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewExpiryFuturesMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-expiry-futures-market [flags]",
		Short: "Submit a proposal to launch expiry futures market",
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

			flagsMapping := cli.FlagsMapping{
				"Title":             cli.Flag{Flag: govcli.FlagTitle},
				"Description":       cli.Flag{Flag: govcli.FlagDescription},
				"Ticker":            cli.Flag{Flag: FlagTicker},
				"QuoteDenom":        cli.Flag{Flag: FlagQuoteDenom},
				"OracleBase":        cli.Flag{Flag: FlagOracleBase},
				"OracleQuote":       cli.Flag{Flag: FlagOracleQuote},
				"OracleScaleFactor": cli.Flag{Flag: FlagOracleScaleFactor},
				"OracleType": cli.Flag{
					Flag: FlagOracleType,
					Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
						oracleType, err := oracletypes.GetOracleType(origV)
						if err != nil {
							return nil, fmt.Errorf("error parsing oracle type: %w", err)
						}
						return fmt.Sprintf("%v", int32(oracleType)), nil
					},
				},
				"Expiry":                 cli.Flag{Flag: FlagExpiry},
				"InitialMarginRatio":     cli.Flag{Flag: FlagInitialMarginRatio},
				"MaintenanceMarginRatio": cli.Flag{Flag: FlagMaintenanceMarginRatio},
				"ReduceMarginRatio":      cli.Flag{Flag: FlagReduceMarginRatio},
				"MakerFeeRate":           cli.Flag{Flag: FlagMakerFeeRate},
				"TakerFeeRate":           cli.Flag{Flag: FlagTakerFeeRate},
				"MinPriceTickSize":       cli.Flag{Flag: FlagMinPriceTickSize},
				"MinQuantityTickSize":    cli.Flag{Flag: FlagMinQuantityTickSize},
				"MinNotional":            cli.Flag{Flag: FlagMinNotional},
				"Admin":                  cli.Flag{Flag: FlagAdmin},
				"AdminPermissions":       cli.Flag{Flag: FlagAdminPermissions},
			}
			argsMapping := cli.ArgsMapping{}

			proposal := exchangev2.ExpiryFuturesMarketLaunchProposal{
				AdminInfo: &exchangev2.AdminInfo{},
			}

			err = cli.ParseFieldsFromFlagsAndArgs(&proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
			if err != nil {
				return err
			}

			message := exchangev2.MsgExpiryFuturesMarketLaunch{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: &proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
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

	cmd.Example = `tx exchange propose-expiry-futures-market \
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
			--expedited=false \
			--deposit="1000000000000000000inj" \
			--from=genesis \
			--keyring-backend=file \
			--yes`

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
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")
	cmd.Flags().String(FlagMinNotional, "0", "min notional")
	cmd.Flags().String(FlagAdmin, "", "market admin")
	cmd.Flags().Uint32(FlagAdminPermissions, 0, "admin permissions level")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
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

			reduceMarginRatio, err := decimalFromFlag(cmd, FlagReduceMarginRatio)
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

			msg := &exchangev2.MsgInstantPerpetualMarketLaunch{
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
				ReduceMarginRatio:      reduceMarginRatio,
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
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
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
			--oracle-scale-factor="0" \
			--maker-fee-rate="0.0005" \
			--taker-fee-rate="0.0012" \
			--expiry="1685460582" \
			--settlement-time="1690730982" \
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

			msg := &exchangev2.MsgInstantBinaryOptionsMarketLaunch{
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
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
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

			msg := &exchangev2.MsgAdminUpdateBinaryOptionsMarket{
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

			msg := &exchangev2.MsgCreateBinaryOptionsLimitOrder{
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

			msg := &exchangev2.MsgCreateBinaryOptionsMarketOrder{
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
			cancelMessage := exchangev2.MsgCancelBinaryOptionsOrder{}

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
			--min-price-tick-size="0.01" \
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

			reduceMarginRatio, err := decimalFromFlag(cmd, FlagReduceMarginRatio)
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

			msg := &exchangev2.MsgInstantExpiryFuturesMarketLaunch{
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
				ReduceMarginRatio:      reduceMarginRatio,
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
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
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
		    --proposal="path/to/trading-reward-campaign-launch-proposal.json" \
			--expedited=false \
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

			proposal, err := parseTradingRewardCampaignLaunchProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgTradingRewardCampaignLaunch{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
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
		    --proposal="path/to/trading-reward-campaign-update-proposal.json" \
			--expedited=false \
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

			proposal, err := parseTradingRewardCampaignUpdateProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgTradingRewardCampaignUpdate{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
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

			proposal, err := parseTradingRewardPointsUpdateProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgTradingRewardPendingPointsUpdate{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
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
			--expedited=false \
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

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			spendMessage := exchangev2.MsgBatchCommunityPoolSpend{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&spendMessage,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
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

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgFeeDiscount{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().
		String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
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
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			batchModificationMessage := exchangev2.MsgBatchExchangeModification{
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

			msg, err := govtypesv1.NewMsgSubmitProposal(
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
		Short: "Submit a proposal to update derivative market params",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			return executeDerivativeMarketParamUpdate(cmd, args, clientCtx)
		},
	}

	cmd.Example = `tx exchange update-derivative-market-params \
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
		--expedited=false \
		--from=genesis \
		--keyring-backend=file \
		--yes`

	setupDerivativeMarketParamUpdateFlags(cmd)

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

			msg := &exchangev2.MsgSubaccountTransfer{
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
			--expedited=false \
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

			proposal, err := forcedMarketSettlementArgsToContent(
				cmd,
				marketID,
				settlementPrice,
			)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgMarketForcedSettlement{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().String(FlagMarketID, "", "ID of market to update params")
	cmd.Flags().String(FlagSettlementPrice, "", "settlement price")
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
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
	--min-quantity-tick-size "1" \
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
				minQuantityTickSize, err = math.LegacyNewDecFromStr(strMinQuantityTickSize)
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

			msg := &exchangev2.MsgUpdateSpotMarket{
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

			strReduceMarginRatio, err := cmd.Flags().GetString(FlagReduceMarginRatio)
			if err != nil {
				return err
			}

			var reduceMarginRatio math.LegacyDec
			if strReduceMarginRatio != "" {
				reduceMarginRatio, err = math.LegacyNewDecFromStr(strReduceMarginRatio)
				if err != nil {
					return err
				}
			}

			msg := &exchangev2.MsgUpdateDerivativeMarket{
				Admin:                     clientCtx.GetFromAddress().String(),
				MarketId:                  common.HexToHash(args[0]).String(),
				NewTicker:                 ticker,
				NewMinPriceTickSize:       minPriceTickSize,
				NewMinQuantityTickSize:    minQuantityTickSize,
				NewInitialMarginRatio:     initialMarginRatio,
				NewMaintenanceMarginRatio: maintenanceMarginRatio,
				NewReduceMarginRatio:      reduceMarginRatio,
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
	cmd.Flags().String(FlagReduceMarginRatio, "", "new reduce margin ratio")

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

			msg := &exchangev2.MsgDeposit{
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

			msg := &exchangev2.MsgWithdraw{
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

			msg := &exchangev2.MsgExternalTransfer{
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
			msg := &exchangev2.MsgRewardsOptOut{
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
			--expedited=false \
			--from=genesis \
			--keyring-backend=file \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			multipliers := make([]*exchangev2.MarketFeeMultiplier, 0)
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
				multiplier := exchangev2.MarketFeeMultiplier{
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

			proposal := &exchangev2.AtomicMarketOrderFeeMultiplierScheduleProposal{
				Title:                title,
				Description:          description,
				MarketFeeMultipliers: multipliers,
			}

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			message := exchangev2.MsgAtomicMarketOrderFeeMultiplierSchedule{
				Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
				Proposal: proposal,
			}
			messages := []sdk.Msg{
				&message,
			}

			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			msg, err := govtypesv1.NewMsgSubmitProposal(
				messages,
				deposit,
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

	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
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

			msg := &exchangev2.MsgAuthorizeStakeGrants{}
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
			msg := exchangev2.MsgActivateStakeGrant{
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

func NewSetDelegationTransferReceiversTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-delegation-transfer-receivers [receivers]",
		Args:  cobra.ExactArgs(1),
		Short: "Set the receivers of the delegation transfer",
		Long: `Set the receivers of the delegation transfer. \

		Example:
		$ %s tx exchange set-delegation-transfer-receivers inj1jcltmuhplrdcwp7stlr4hlhlhgd4htqhe4c0cs,inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c
			--yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee := clientCtx.GetFromAddress()
			receivers := strings.Split(args[0], ",")
			msg := &exchangev2.MsgSetDelegationTransferReceivers{
				Sender:    grantee.String(),
				Receivers: receivers,
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

func NewIncreasePositionMarginTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"increase-position-margin <source-subaccount-id> <dest-subaccount-id> <market-id> <amount>",
		"Increase margin in an open position",
		&exchangev2.MsgIncreasePositionMargin{},
		nil,
		cli.ArgsMapping{
			"SourceSubaccountId":      cli.Arg{Index: 0},
			"DestinationSubaccountId": cli.Arg{Index: 1},
			"MarketId":                cli.Arg{Index: 2, Transform: getDerivativeMarketIdFromTicker},
			"Amount":                  cli.Arg{Index: 3},
		},
	)
	cmd.Example = `injectived tx exchange increase-position-margin \
	0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 \
	0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 \
	"ETH/USDT PERP" \
	10`
	return cmd
}

func NewDecreasePositionMarginTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"decrease-position-margin <source-subaccount-id> <dest-subaccount-id> <market-id> <amount>",
		"Decrease margin in an open position",
		&exchangev2.MsgDecreasePositionMargin{},
		nil,
		cli.ArgsMapping{
			"SourceSubaccountId":      cli.Arg{Index: 0},
			"DestinationSubaccountId": cli.Arg{Index: 1},
			"MarketId":                cli.Arg{Index: 2, Transform: getDerivativeMarketIdFromTicker},
			"Amount":                  cli.Arg{Index: 3},
		},
	)
	cmd.Example = `injectived tx exchange decrease-position-margin \
	0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 \
	0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 \
	"ETH/USDT PERP" \
	10`
	return cmd
}

func NewMsgLiquidatePositionTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidate-position <subaccount-id> <market-id>",
		Args:  cobra.ExactArgs(2),
		Short: "Liquidate a position",
		Long: `Liquidate a position

		Example:
		$ %s tx exchange liquidate-position \
		0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000 \
		0x77261d2236f465ca70995043e4134897bcf8aee1262ba69d93ad819d5722cd6a
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

func NewCancelPostOnlyModeTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"cancel-post-only-mode",
		"Cancel the post-only mode if it's currently active",
		&exchangev2.MsgCancelPostOnlyMode{},
		nil,
		cli.ArgsMapping{},
	)
	cmd.Example = `injectived tx exchange cancel-post-only-mode \
		--from=genesis \
		--keyring-backend=file \
		--yes`
	return cmd
}

func getDerivativeMarketParamUpdateFlagsMapping() cli.FlagsMapping {
	return cli.FlagsMapping{
		"Title":                  cli.Flag{Flag: govcli.FlagTitle},
		"Description":            cli.Flag{Flag: govcli.FlagDescription},
		"MarketId":               cli.Flag{Flag: FlagMarketID},
		"InitialMarginRatio":     cli.Flag{Flag: FlagInitialMarginRatio},
		"MaintenanceMarginRatio": cli.Flag{Flag: FlagMaintenanceMarginRatio},
		"ReduceMarginRatio":      cli.Flag{Flag: FlagReduceMarginRatio},
		"MakerFeeRate":           cli.Flag{Flag: FlagMakerFeeRate},
		"TakerFeeRate":           cli.Flag{Flag: FlagTakerFeeRate},
		"RelayerFeeShareRate":    cli.Flag{Flag: FlagRelayerFeeShareRate},
		"MinPriceTickSize":       cli.Flag{Flag: FlagMinPriceTickSize},
		"MinQuantityTickSize":    cli.Flag{Flag: FlagMinQuantityTickSize},
		"HourlyInterestRate":     cli.Flag{Flag: FlagHourlyInterestRate},
		"HourlyFundingRateCap":   cli.Flag{Flag: FlagHourlyFundingRateCap},
		"Status": cli.Flag{
			Flag: FlagMarketStatus,
			Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
				var status exchangev2.MarketStatus
				if origV != "" {
					newStatus, ok := exchangev2.MarketStatus_value[origV]
					if !ok {
						return nil, fmt.Errorf("incorrect market status: %s", origV)
					}
					status = exchangev2.MarketStatus(newStatus)
				} else {
					status = exchangev2.MarketStatus_Unspecified
				}
				return fmt.Sprintf("%v", int32(status)), nil
			},
		},
		"OracleBase":        cli.Flag{Flag: FlagOracleBase},
		"OracleQuote":       cli.Flag{Flag: FlagOracleQuote},
		"OracleScaleFactor": cli.Flag{Flag: FlagOracleScaleFactor},
		"OracleType": cli.Flag{
			Flag: FlagOracleType,
			Transform: func(origV string, _ grpc.ClientConn) (transformedV any, err error) {
				oracleType, err := oracletypes.GetOracleType(origV)
				if err != nil {
					return nil, fmt.Errorf("error parsing oracle type: %w", err)
				}
				return fmt.Sprintf("%v", int32(oracleType)), nil
			},
		},
		"Ticker":           cli.Flag{Flag: FlagTicker},
		"MinNotional":      cli.Flag{Flag: FlagMinNotional},
		"Admin":            cli.Flag{Flag: FlagAdmin},
		"AdminPermissions": cli.Flag{Flag: FlagAdminPermissions},
		"BaseDecimals":     cli.Flag{Flag: FlagBaseDecimals},
		"QuoteDecimals":    cli.Flag{Flag: FlagQuoteDecimals},
	}
}

func setupDerivativeMarketParamUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagMarketID, "", "ID of market to update params")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(FlagReduceMarginRatio, "", "reduce margin ratio")
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
	cmd.Flags().Bool(FlagExpedited, false, "set the expedited value for the governance proposal")
	cliflags.AddGovProposalFlags(cmd)
	cliflags.AddTxFlagsToCmd(cmd)
}

func createDerivativeMarketParamUpdateProposal(
	cmd *cobra.Command, args []string, clientCtx client.Context,
) (*exchangev2.DerivativeMarketParamUpdateProposal, error) {
	flagsMapping := getDerivativeMarketParamUpdateFlagsMapping()
	argsMapping := cli.ArgsMapping{}

	proposal := &exchangev2.DerivativeMarketParamUpdateProposal{
		OracleParams: &exchangev2.OracleParams{},
		AdminInfo:    &exchangev2.AdminInfo{},
	}

	err := cli.ParseFieldsFromFlagsAndArgs(proposal, flagsMapping, argsMapping, cmd.Flags(), args, clientCtx)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

func createDerivativeMarketParamUpdateMessage(proposal *exchangev2.DerivativeMarketParamUpdateProposal) sdk.Msg {
	return &exchangev2.MsgDerivativeMarketParamUpdate{
		Sender:   authtypes.NewModuleAddress(govgeneraltypes.ModuleName).String(),
		Proposal: proposal,
	}
}

func executeDerivativeMarketParamUpdate(cmd *cobra.Command, args []string, clientCtx client.Context) error {
	depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
	if err != nil {
		return err
	}

	amount, err := sdk.ParseCoinsNormalized(depositStr)
	if err != nil {
		return err
	}

	proposal, err := createDerivativeMarketParamUpdateProposal(cmd, args, clientCtx)
	if err != nil {
		return err
	}

	message := createDerivativeMarketParamUpdateMessage(proposal)
	messages := []sdk.Msg{message}

	expedited, err := cmd.Flags().GetBool(FlagExpedited)
	if err != nil {
		return err
	}

	msg, err := govtypesv1.NewMsgSubmitProposal(
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
}

func getSpotMarketIdFromTicker(ticker string, ctx grpc.ClientConn) (any, error) {
	queryClient := exchangev2.NewQueryClient(ctx)
	req := &exchangev2.QuerySpotMarketsRequest{
		Status: "Active",
	}
	res, err := queryClient.SpotMarkets(context.Background(), req)
	if err != nil {
		return nil, err
	}
	var market *exchangev2.SpotMarket
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
	queryClient := exchangev2.NewQueryClient(ctx)
	req := &exchangev2.QueryDerivativeMarketsRequest{}
	res, err := queryClient.DerivativeMarkets(context.Background(), req)
	if err != nil {
		return nil, err
	}
	var market *exchangev2.DerivativeMarket
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

func forcedMarketSettlementArgsToContent(
	cmd *cobra.Command, marketID string,
	settlementPrice *math.LegacyDec,
) (*exchangev2.MarketForcedSettlementProposal, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := exchangev2.NewMarketForcedSettlementProposal(
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

	denomDecimals := make([]*exchangev2.DenomDecimals, 0, len(denoms))
	for idx, denom := range denoms {
		denomDecimals = append(denomDecimals, &exchangev2.DenomDecimals{
			Denom:    denom,
			Decimals: uint64(decimals[idx]),
		})
	}

	content := exchangev2.NewUpdateDenomDecimalsProposal(
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

func buildExchangeAuthz(
	subaccountId string,
	marketIDs []string,
	msgType string,
) authz.Authorization {
	switch msgType {
	// spot messages
	case "MsgCreateSpotLimitOrder":
		return &exchangev2.CreateSpotLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCreateSpotMarketOrder":
		return &exchangev2.CreateSpotMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCreateSpotLimitOrders":
		return &exchangev2.BatchCreateSpotLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCancelSpotOrder":
		return &exchangev2.CancelSpotOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCancelSpotOrders":
		return &exchangev2.BatchCancelSpotOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}

	// derivative messages
	case "MsgCreateDerivativeLimitOrder":
		return &exchangev2.CreateDerivativeLimitOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCreateDerivativeMarketOrder":
		return &exchangev2.CreateDerivativeMarketOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCreateDerivativeLimitOrders":
		return &exchangev2.BatchCreateDerivativeLimitOrdersAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgCancelDerivativeOrder":
		return &exchangev2.CancelDerivativeOrderAuthz{
			SubaccountId: subaccountId,
			MarketIds:    marketIDs,
		}
	case "MsgBatchCancelDerivativeOrders":
		return &exchangev2.BatchCancelDerivativeOrdersAuthz{
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
	return &exchangev2.BatchUpdateOrdersAuthz{
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
	cmd.Flags().String(FlagExpirationBlock, "0", "expiration block")
	cliflags.AddTxFlagsToCmd(cmd)
}
