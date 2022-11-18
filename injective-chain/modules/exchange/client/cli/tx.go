package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzcli "github.com/cosmos/cosmos-sdk/x/authz/client/cli"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/version"
)

// flags for exchange tx commands
const (
	FlagMarketID                = "market-id"
	FlagMarketIDs               = "market-ids"
	FlagOrderType               = "order-type"
	FlagSubaccountID            = "subaccount-id"
	FlagPrice                   = "price"
	FlagQuantity                = "quantity"
	FlagMargin                  = "margin"
	FlagTicker                  = "ticker"
	FlagBaseDenom               = "base-denom"
	FlagQuoteDenom              = "quote-denom"
	FlagOracleSymbol            = "oracle-symbol"
	FlagOracleProvider          = "oracle-provider"
	FlagAdmin                   = "admin"
	FlagOracleBase              = "oracle-base"
	FlagOracleQuote             = "oracle-quote"
	FlagOracleScaleFactor       = "oracle-scale-factor"
	FlagOracleType              = "oracle-type"
	FlagExpiry                  = "expiry"
	FlagSettlementTime          = "settlement-time"
	FlagInitialMarginRatio      = "initial-margin-ratio"
	FlagMaintenanceMarginRatio  = "maintenance-margin-ratio"
	FlagMakerFeeRate            = "maker-fee-rate"
	FlagTakerFeeRate            = "taker-fee-rate"
	FlagRelayerFeeShareRate     = "relayer-fee-share-rate"
	FlagHourlyInterestRate      = "hourly-interest-rate"
	FlagHourlyFundingRateCap    = "hourly-funding-rate-cap"
	FlagMinPriceTickSize        = "min-price-tick-size"
	FlagMinQuantityTickSize     = "min-quantity-tick-size"
	FlagMarketStatus            = "market-status"
	FlagTradingRewardCampaigns  = "campaigns"
	FlagSpotMarketIDs           = "spot-market-ids"
	FlagSpotMarketWeights       = "spot-market-weights"
	FlagDerivativeMarketIDs     = "derivative-market-ids"
	FlagDerivativeMarketWeights = "derivative-market-weights"
	FlagMaxEpochRewards         = "max-epoch-rewards"
	FlagFeeRecipient            = "fee-recipient"
	FlagTriggerPrice            = "trigger-price"
	FlagReduceOnly              = "reduce-only"
	FlagOrderHash               = "order-hash"
	FlagSettlementPrice         = "settlement-price"
	FlagDecimals                = "decimals"
	FlagDenom                   = "denom"
	FlagDenoms                  = "denoms"
	FlagExpirationTime          = "expiration-time"
)

// NewTxCmd returns a root CLI command handler for certain modules/exchange transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Exchange transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewExchangeEnableProposalTxCmd(),
		NewSpotMarketUpdateParamsProposalTxCmd(),
		NewSpotMarketLaunchProposalTxCmd(),
		NewPerpetualMarketLaunchProposalTxCmd(),
		NewExpiryFuturesMarketLaunchProposalTxCmd(),
		NewInstantSpotMarketLaunchTxCmd(),
		NewInstantPerpetualMarketLaunchTxCmd(),
		NewInstantBinaryOptionsMarketLaunchTxCmd(),
		NewAdminUpdateBinaryOptionsMarketTxCmd(),
		NewInstantExpiryFuturesMarketLaunchTxCmd(),
		NewDerivativeMarketParamUpdateProposalTxCmd(),
		NewBatchExchangeModificationProposalTxCmd(),
		TradingRewardCampaignLaunchProposalTxCmd(),
		TradingRewardCampaignUpdateProposalTxCmd(),
		TradingRewardPointsUpdateProposalTxCmd(),
		FeeDiscountProposalTxCmd(),
		BatchCommunityPoolSpendProposalTxCmd(),
		NewDepositTxCmd(),
		NewWithdrawTxCmd(),
		NewSubaccountTransferTxCmd(),
		NewExternalTransferTxCmd(),
		NewCreateSpotLimitOrderTxCmd(),
		NewCreateSpotMarketOrderTxCmd(),
		NewSubscribeToSpotVaultTxCmd(),
		NewRedeemFromSpotVaultTxCmd(),
		NewSubscribeToDerivativeVaultTxCmd(),
		NewRedeemFromDerivativeVaultTxCmd(),
		NewCancelSpotLimitOrderTxCmd(),
		NewCancelDerivativeLimitOrderTxCmd(),
		NewCreateDerivativeLimitOrderTxCmd(),
		NewCreateDerivativeMarketOrderTxCmd(),
		NewAuthzTxCmd(),
		NewBatchUpdateAuthzTxCmd(),
		NewRewardsOptOutTxCmd(),
		NewCreateBinaryOptionsMarketOrderTxCmd(),
		NewCreateBinaryOptionsLimitOrderTxCmd(),
		NewCancelBinaryOptionsOrderTxCmd(),
		NewMarketForcedSettlementTxCmd(),
		NewAtomicMarketOrderFeeMultiplierScheduleProposalTxCmd(),
	)
	return txCmd
}

func NewInstantSpotMarketLaunchTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instant-spot-market-launch [ticker] [base_denom] [quote_denom] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Launch spot market by paying listing fee without governance",
		Long: `Launch spot market by paying listing fee without governance.

		Example:
		$ %s tx exchange instant-spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=1000000000 \
			--min-quantity-tick-size=1000000000000000 \
			--from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
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

			msg := &types.MsgInstantSpotMarketLaunch{
				Sender:              clientCtx.GetFromAddress().String(),
				Ticker:              args[0],
				BaseDenom:           args[1],
				QuoteDenom:          args[2],
				MinPriceTickSize:    minPriceTickSize,
				MinQuantityTickSize: minQuantityTickSize,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCreateSpotLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-spot-limit-order [order_type] [market_ticker] [quantity] [price] [flags]",
		Args:  cobra.ExactArgs(4),
		Short: "Create Spot Limit Order",
		Long: `Create Spot Limit Order.

		Example:
		$ %s tx exchange create-spot-limit-order buy ETH/USDT 2.4 2000.1 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var orderType types.OrderType
			switch args[0] {
			case "buy":
				orderType = types.OrderType_BUY
			case "sell":
				orderType = types.OrderType_SELL
			case "buy-PO":
				orderType = types.OrderType_BUY_PO
			case "sell-PO":
				orderType = types.OrderType_SELL_PO
			default:
				return errors.New(`order type must be "buy", "sell", "buy-PO" or "sell-PO"`)
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QuerySpotMarketsRequest{
				Status: "Active",
			}
			res, err := queryClient.SpotMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			ticker := args[1]
			var market *types.SpotMarket
			for _, spotMarket := range res.Markets {
				if spotMarket.Ticker == ticker {
					market = spotMarket
				}
			}

			if market == nil {
				return errors.New("Spot Market of ticker " + ticker + " not found")
			}

			quantity, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}
			price, err := sdk.NewDecFromStr(args[3])
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())
			subaccountID := types.EthAddressToSubaccountID(ethAddress)

			msg := &types.MsgCreateSpotLimitOrder{
				Sender: fromAddress,
				Order: types.SpotOrder{
					MarketId: market.MarketId,
					OrderInfo: types.OrderInfo{
						SubaccountId: subaccountID.Hex(),
						FeeRecipient: fromAddress,
						Price:        price,
						Quantity:     quantity,
					},
					OrderType:    orderType,
					TriggerPrice: nil,
				},
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

func NewCreateSpotMarketOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-spot-market-order [direction] [market_ticker] [quantity] [worst_price] [flags]",
		Args:  cobra.ExactArgs(4),
		Short: "Create Spot Market Order",
		Long: `Create Spot Market Order.

		Example:
		$ %s tx exchange create-spot-market-order buy ETH/USDT 2.4 2030.1 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var orderType types.OrderType
			switch args[0] {
			case "buy":
				orderType = types.OrderType_BUY
			case "sell":
				orderType = types.OrderType_SELL
			default:
				return errors.New("order type must be \"buy\" or \"sell\"")
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QuerySpotMarketsRequest{
				Status: "Active",
			}
			res, err := queryClient.SpotMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			ticker := args[1]
			var market *types.SpotMarket
			for _, spotMarket := range res.Markets {
				if spotMarket.Ticker == ticker {
					market = spotMarket
				}
			}

			if market == nil {
				return errors.New("Spot Market of ticker " + ticker + " not found")
			}

			quantity, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}
			price, err := sdk.NewDecFromStr(args[3])
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())
			subaccountID := types.EthAddressToSubaccountID(ethAddress)

			msg := &types.MsgCreateSpotMarketOrder{
				Sender: fromAddress,
				Order: types.SpotOrder{
					MarketId: market.MarketId,
					OrderInfo: types.OrderInfo{
						SubaccountId: subaccountID.Hex(),
						FeeRecipient: fromAddress,
						Price:        price,
						Quantity:     quantity,
					},
					OrderType:    orderType,
					TriggerPrice: nil,
				},
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

type VaultSubscribeArgs struct {
	VaultSubaccountId      string   `json:"vault_subaccount_id"`
	SubscriberSubaccountId string   `json:"subscriber_subaccount_id"`
	MarginRatio            *sdk.Dec `json:"margin_ratio,omitempty"`
}
type VaultSubscribe struct {
	SubscribeArgs VaultSubscribeArgs `json:"args"`
}

type VaultRedeemArgs struct {
	VaultSubaccountId    string  `json:"vault_subaccount_id"`
	RedeemerSubaccountId string  `json:"redeemer_subaccount_id"`
	LpTokenBurnAmount    sdk.Int `json:"lp_token_burn_amount"`
}
type VaultRedeem struct {
	RedeemArgs VaultRedeemArgs `json:"args"`
}

type VaultInput struct {
	Subscribe *VaultSubscribe `json:"Subscribe,omitempty"`
	Redeem    *VaultRedeem    `json:"Redeem,omitempty"`
}

func NewSubscribeToSpotVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe-to-spot-vault [flags]",
		Args:  cobra.ExactArgs(4),
		Short: "subscribe-to-spot-vault",
		Long: `subscribe-to-spot-vault.

		Example:
		$ %s tx exchange subscribe-to-spot-vault [vault subaccount id] [subscriber subaccount id] [quote amount] [base amount] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			subscriberSubaccountId := args[1]

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryMarketIDFromVaultRequest{
				VaultSubaccountId: vaultSubaccountId,
			}
			res, err := queryClient.QueryMarketIDFromVault(context.Background(), req)
			if err != nil {
				return err
			}

			marketRequest := &types.QuerySpotMarketRequest{
				MarketId: res.MarketId,
			}
			marketResp, err := queryClient.SpotMarket(context.Background(), marketRequest)
			if err != nil {
				return err
			}
			quoteDenom := marketResp.Market.QuoteDenom
			baseDenom := marketResp.Market.BaseDenom

			quoteAmount, ok := sdk.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[2])
			}
			baseAmount, ok := sdk.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[3])
			}

			bankFunds := sdk.NewCoins(sdk.NewCoin(quoteDenom, quoteAmount), sdk.NewCoin(baseDenom, baseAmount))
			fromAddress := clientCtx.GetFromAddress().String()

			vaultSubscribeArgs := VaultSubscribeArgs{
				MarginRatio:            nil,
				VaultSubaccountId:      vaultSubaccountId,
				SubscriberSubaccountId: subscriberSubaccountId,
			}
			vaultSubscribe := VaultSubscribe{
				SubscribeArgs: vaultSubscribeArgs,
			}
			vaultInput := VaultInput{
				Subscribe: &vaultSubscribe,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultSubscribe",
				Args:   vaultInput,
			}
			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				fmt.Println("err", err)
				return err
			}

			vaultAccount := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           bankFunds.String(),
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
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

func NewRedeemFromSpotVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-from-spot-vault [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "redeem-from-spot-vault",
		Long: `redeem-from-spot-vault.

		Example:
		$ %s tx exchange redeem-from-spot-vault [vault subaccount id] [redeemer subaccount id] [lp token burn amount] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			redeemerSubaccountId := args[1]
			lpTokenBurnAmount, ok := sdk.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[2])
			}

			fromAddress := clientCtx.GetFromAddress().String()

			vaultRedeemArgs := VaultRedeemArgs{
				LpTokenBurnAmount:    lpTokenBurnAmount,
				VaultSubaccountId:    vaultSubaccountId,
				RedeemerSubaccountId: redeemerSubaccountId,
			}
			vaultRedeem := VaultRedeem{
				RedeemArgs: vaultRedeemArgs,
			}
			vaultInput := VaultInput{
				Redeem: &vaultRedeem,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultRedeem",
				Args:   vaultInput,
			}
			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				fmt.Println("err", err)
				return err
			}

			vaultAccount := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           "",
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
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

func NewSubscribeToDerivativeVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe-to-derivative-vault [flags]",
		Args:  cobra.ExactArgs(4),
		Short: "subscribe-to-derivative-vault",
		Long: `subscribe-to-derivative-vault.

		Example:
		$ %s tx exchange subscribe-to-derivative-vault [vault subaccount id] [subscriber subaccount id] [margin ratio] [amount] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			subscriberSubaccountId := args[1]
			subscriberMarginRatio, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryMarketIDFromVaultRequest{
				VaultSubaccountId: vaultSubaccountId,
			}
			res, err := queryClient.QueryMarketIDFromVault(context.Background(), req)
			if err != nil {
				return err
			}

			marketRequest := &types.QueryDerivativeMarketRequest{
				MarketId: res.MarketId,
			}
			marketResp, err := queryClient.DerivativeMarket(context.Background(), marketRequest)
			if err != nil {
				return err
			}
			quoteDenom := marketResp.Market.Market.QuoteDenom

			amount, ok := sdk.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[3])
			}

			bankFunds := sdk.NewCoins(sdk.NewCoin(quoteDenom, amount))
			fromAddress := clientCtx.GetFromAddress().String()

			vaultSubscribeArgs := VaultSubscribeArgs{
				MarginRatio:            &subscriberMarginRatio,
				VaultSubaccountId:      vaultSubaccountId,
				SubscriberSubaccountId: subscriberSubaccountId,
			}
			vaultSubscribe := VaultSubscribe{
				SubscribeArgs: vaultSubscribeArgs,
			}
			vaultInput := VaultInput{
				Subscribe: &vaultSubscribe,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultSubscribe",
				Args:   vaultInput,
			}
			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				fmt.Println("err", err)
				return err
			}

			vaultAccount := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           bankFunds.String(),
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
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

func NewRedeemFromDerivativeVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-from-derivative-vault [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "redeem-from-derivative-vault",
		Long: `redeem-from-derivative-vault.

		Example:
		$ %s tx exchange redeem-from-derivative-vault [vault subaccount id] [redeemer subaccount id] [lp token burn amount] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			redeemerSubaccountId := args[1]
			lpTokenBurnAmount, ok := sdk.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[2])
			}

			fromAddress := clientCtx.GetFromAddress().String()

			vaultRedeemArgs := VaultRedeemArgs{
				LpTokenBurnAmount:    lpTokenBurnAmount,
				VaultSubaccountId:    vaultSubaccountId,
				RedeemerSubaccountId: redeemerSubaccountId,
			}
			vaultRedeem := VaultRedeem{
				RedeemArgs: vaultRedeemArgs,
			}
			vaultInput := VaultInput{
				Redeem: &vaultRedeem,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultRedeem",
				Args:   vaultInput,
			}
			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				fmt.Println("err", err)
				return err
			}

			vaultAccount := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           "",
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
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

func NewCancelSpotLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-spot-limit-order [market_ticker] [order_hash] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Cancel Spot Limit Order",
		Long: `Cancel Spot Limit Order.

		Example:
		$ %s tx exchange cancel-spot-limit-order ETH/USDT 0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QuerySpotMarketsRequest{
				Status: "Active",
			}
			res, err := queryClient.SpotMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			ticker := args[0]
			var market *types.SpotMarket
			for _, spotMarket := range res.Markets {
				if spotMarket.Ticker == ticker {
					market = spotMarket
				}
			}

			if market == nil {
				return errors.New("Spot Market of ticker " + ticker + " not found")
			}

			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())
			subaccountID := types.EthAddressToSubaccountID(ethAddress)

			msg := &types.MsgCancelSpotOrder{
				Sender:       fromAddress,
				MarketId:     market.MarketId,
				SubaccountId: subaccountID.Hex(),
				OrderHash:    args[1],
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

func NewCancelDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-derivative-limit-order [market_ticker] [order_hash] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Cancel Derivative Limit Order",
		Long: `Cancel Derivative Limit Order.

		Example:
		$ %s tx exchange cancel-derivative-limit-order ETH/USDT 0xc66d1e52aa24d16eaa8eb0db773ab019e82daf96c14af0e105a175db22cd0fc8 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryDerivativeMarketsRequest{}
			res, err := queryClient.DerivativeMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			ticker := args[0]
			var market *types.DerivativeMarket
			for _, derivativeMarket := range res.Markets {
				if derivativeMarket.Market.Ticker == ticker {
					market = derivativeMarket.Market
				}
			}

			if market == nil {
				return errors.New("Derivative Market of ticker " + ticker + " not found")
			}

			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())
			subaccountID := types.EthAddressToSubaccountID(ethAddress)

			msg := &types.MsgCancelDerivativeOrder{
				Sender:       fromAddress,
				MarketId:     market.MarketId,
				SubaccountId: subaccountID.Hex(),
				OrderHash:    args[1],
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

func NewCreateDerivativeLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-derivative-limit-order [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create Derivative Limit Order",
		Long: `Create Derivative Limit Order.

		Example:
		$ %s tx exchange create-derivative-limit-order \
			--order-type="buy" \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--price="4.1" \
			--quantity="10.01" \
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

			marketID, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			orderTypeStr, err := cmd.Flags().GetString(FlagOrderType)
			if err != nil {
				return err
			}

			var orderType types.OrderType

			switch orderTypeStr {
			case "buy":
				orderType = types.OrderType_BUY
			case "sell":
				orderType = types.OrderType_SELL
			case "buy-PO":
				orderType = types.OrderType_BUY_PO
			case "sell-PO":
				orderType = types.OrderType_SELL_PO
			default:
				return errors.New(`order type must be "buy", "sell", "buy-PO" or "sell-PO"`)
			}

			quantity, err := cmd.Flags().GetString(FlagQuantity)
			if err != nil {
				return err
			}

			price, err := cmd.Flags().GetString(FlagPrice)
			if err != nil {
				return err
			}

			margin, err := cmd.Flags().GetString(FlagMargin)
			if err != nil {
				return err
			}
			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())

			subaccountID, err := cmd.Flags().GetString(FlagSubaccountID)
			if subaccountID == "" {
				subaccountID = types.EthAddressToSubaccountID(ethAddress).Hex()
			} else if err != nil {
				return err
			}

			msg := &types.MsgCreateDerivativeLimitOrder{
				Sender: fromAddress,
				Order: types.DerivativeOrder{
					MarketId: marketID,
					OrderInfo: types.OrderInfo{
						SubaccountId: subaccountID,
						FeeRecipient: fromAddress,
						Price:        sdk.MustNewDecFromStr(price),
						Quantity:     sdk.MustNewDecFromStr(quantity),
					},
					Margin:       sdk.MustNewDecFromStr(margin),
					OrderType:    orderType,
					TriggerPrice: nil,
				},
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCreateDerivativeMarketOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-derivative-market-order [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create Derivative Market Order",
		Long: `Create Derivative Market Order.

		Example:
		$ %s tx exchange create-derivative-market-order \
			--order-type="buy" \
			--market-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--subaccount-id="0x17d9b5fb67666df72a5a858eb9b81104b99da760e3036a8243e05532d50e1c7c" \
			--price="4.1" \
			--quantity="10.01" \
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

			marketID, err := cmd.Flags().GetString(FlagMarketID)
			if err != nil {
				return err
			}

			orderTypeStr, err := cmd.Flags().GetString(FlagOrderType)
			if err != nil {
				return err
			}

			var orderType types.OrderType
			switch orderTypeStr {
			case "buy":
				orderType = types.OrderType_BUY
			case "sell":
				orderType = types.OrderType_SELL
			default:
				return errors.New("order type must be \"buy\" or \"sell\"")
			}

			quantity, err := cmd.Flags().GetString(FlagQuantity)
			if err != nil {
				return err
			}

			price, err := cmd.Flags().GetString(FlagPrice)
			if err != nil {
				return err
			}

			margin, err := cmd.Flags().GetString(FlagMargin)
			if err != nil {
				return err
			}
			fromAddress := clientCtx.GetFromAddress().String()

			senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
			if err != nil {
				return errors.Wrap(err, "must provide a valid Bech32 address")
			}
			ethAddress := common.BytesToAddress(senderAddr.Bytes())

			subaccountID, err := cmd.Flags().GetString(FlagSubaccountID)
			if subaccountID == "" {
				subaccountID = types.EthAddressToSubaccountID(ethAddress).Hex()
			} else if err != nil {
				return err
			}

			msg := &types.MsgCreateDerivativeMarketOrder{
				Sender: fromAddress,
				Order: types.DerivativeOrder{
					MarketId: marketID,
					OrderInfo: types.OrderInfo{
						SubaccountId: subaccountID,
						FeeRecipient: fromAddress,
						Price:        sdk.MustNewDecFromStr(price),
						Quantity:     sdk.MustNewDecFromStr(quantity),
					},
					Margin:       sdk.MustNewDecFromStr(margin),
					OrderType:    orderType,
					TriggerPrice: nil,
				},
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketID, "", "Derivative market ID")
	cmd.Flags().String(FlagOrderType, "", "Order type")
	cmd.Flags().String(FlagSubaccountID, "", "Subaccount ID")
	cmd.Flags().String(FlagPrice, "", "Price of the order")
	cmd.Flags().String(FlagQuantity, "", "Quantity of the order")
	cmd.Flags().String(FlagMargin, "", "Margin for the order")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSpotMarketUpdateParamsProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-spot-market-params [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to update spot market params",
		Long: `Submit a proposal to update spot market params.

		Example:
		$ %s tx exchange update-spot-market-params --market-id="0xacdd4f9cb90ecf5c4e254acbf65a942f562ca33ba718737a93e5cb3caadec3aa" --title="Spot market params update" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
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

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			var (
				makerFeeRate        *sdk.Dec
				takerFeeRate        *sdk.Dec
				relayerFeeShareRate *sdk.Dec
				minPriceTickSize    *sdk.Dec
				minQuantityTickSize *sdk.Dec
				marketStatus        *types.MarketStatus
			)

			makerFeeRateDec, _ := decimalFromFlag(cmd, FlagMakerFeeRate)
			takerFeeRateDec, _ := decimalFromFlag(cmd, FlagTakerFeeRate)
			relayerFeeShareRateDec, _ := decimalFromFlag(cmd, FlagRelayerFeeShareRate)

			minPriceTickSizeStr, _ := cmd.Flags().GetString(FlagMinPriceTickSize)

			minQuantityTickSizeStr, _ := cmd.Flags().GetString(FlagMinQuantityTickSize)

			marketStatusStr, _ := cmd.Flags().GetString(FlagMarketStatus)

			if !makerFeeRateDec.IsNil() {
				makerFeeRate = &makerFeeRateDec
			}

			if !takerFeeRateDec.IsNil() {
				takerFeeRate = &takerFeeRateDec
			}

			if relayerFeeShareRateDec.IsNil() {
				relayerFeeShareRate = nil
			}

			if minPriceTickSizeStr != "" {
				m, err := sdk.NewDecFromStr(minPriceTickSizeStr)
				if err != nil {
					return err
				}
				minPriceTickSize = &m
			}

			if minQuantityTickSizeStr != "" {
				m, err := sdk.NewDecFromStr(minQuantityTickSizeStr)
				if err != nil {
					return err
				}
				minQuantityTickSize = &m
			}

			if marketStatusStr != "" {
				m := types.MarketStatus(types.MarketStatus_value[marketStatusStr])
				marketStatus = &m
			}

			var status types.MarketStatus

			if marketStatus == nil {
				status = types.MarketStatus_Unspecified
			} else {
				status = *marketStatus
			}
			content := types.NewSpotMarketParamUpdateProposal(
				title,
				description,
				common.HexToHash(marketID),
				makerFeeRate,
				takerFeeRate,
				relayerFeeShareRate,
				minPriceTickSize,
				minQuantityTickSize,
				status,
			)

			fmt.Println(content.String())

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

	cmd.Flags().String(FlagMarketID, "", "Spot market ID")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagRelayerFeeShareRate, "", "relayer fee share rate")
	cmd.Flags().String(FlagMinPriceTickSize, "", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "min quantity tick size")
	cmd.Flags().String(FlagMarketStatus, "", "market status")

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSpotMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-market-launch [ticker] [base_denom] [quote_denom] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit a proposal to launch spot-market",
		Long: `Submit a proposal to launch spot-market.

		Example:
		$ %s tx exchange spot-market-launch INJ/ATOM uinj uatom \
			--min-price-tick-size=1000000000 \
			--min-quantity-tick-size=1000000000000000 \
			--maker-fee-rate="0.001" \
			--taker-fee-rate="0.001" \
			--title="INJ/ATOM spot market" \
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

			minPriceTickSize, err := decimalFromFlag(cmd, FlagMinPriceTickSize)
			if err != nil {
				return err
			}

			minQuantityTickSize, err := decimalFromFlag(cmd, FlagMinQuantityTickSize)
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

			content, err := spotMarketLaunchArgsToContent(cmd, args[0], args[1], args[2], minPriceTickSize, minQuantityTickSize, makerFeeRate, takerFeeRate)
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
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewExchangeEnableProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-exchange-enable [exchange-type] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to enable spot or derivatives exchange (exchangeType of spot or derivatives)",
		Long: `Submit a proposal to enable spot or derivatives exchange (exchangeType of spot or derivatives).

		Example:
		$ %s tx exchange spot \
			--title="Enable Spot Exchange" \
			--description="Enable Spot Exchange" \
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

			var exchangeType types.ExchangeType
			switch args[0] {
			case "spot":
				exchangeType = types.ExchangeType_SPOT
			case "derivatives":
				exchangeType = types.ExchangeType_DERIVATIVES
			default:
				return types.ErrBadField
			}

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return err
			}

			content := &types.ExchangeEnableProposal{
				Title:        title,
				Description:  description,
				ExchangeType: exchangeType,
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

func NewPerpetualMarketLaunchProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-perpetual-market [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to launch perpetual market",
		Long: `Submit a proposal to launch perpetual market.

		Example:
		$ %s tx exchange propose-perpetual-market
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

			content, err := perpetualMarketLaunchArgsToContent(
				cmd,
				ticker,
				quoteDenom,
				oracleBase,
				oracleQuote,
				oracleScaleFactor,
				oracleType,
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
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagInitialMarginRatio, "", "initial margin ratio")
	cmd.Flags().String(FlagMaintenanceMarginRatio, "", "maintenance margin ratio")
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().String(FlagMinPriceTickSize, "0.01", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "0.01", "min quantity tick size")

	cliflags.AddTxFlagsToCmd(cmd)
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
	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

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
	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

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
	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")

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

	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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

	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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
	cmd.Flags().String(govcli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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
		Use:   "deposit [amount] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit message to transfer coins from the sender's bank balance into the default subaccount's exchange deposits.",
		Long: `Submit message to transfer coins from the sender's bank balance into the default subaccount's exchange deposits.

		Example:
		$ %s tx exchange deposit 10000inj --from=genesis --keyring-backend=file --yes
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

			msg := &types.MsgDeposit{
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
				expirationDate,
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
			authorization := buildBatchUpdateExchangeAuthz(subAccountId, spotMarketIds, derivativeMarketIds)
			msg, err := authz.NewMsgGrant(
				clientCtx.GetFromAddress(),
				grantee,
				authorization,
				expirationDate,
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
					return types.ErrInvalidArgument.Wrapf("%v does not match a pattern marketId:multiplier", arg)
				}
				marketId := split[0]
				common.HexToHash(marketId)
				feeMultiplier, err := sdk.NewDecFromStr(split[1])
				if err != nil {
					return err
				}
				multiplier := types.MarketFeeMultiplier{MarketId: marketId, FeeMultiplier: feeMultiplier}
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

func spotMarketLaunchArgsToContent(cmd *cobra.Command, ticker, baseDenom, quoteDenom string, minPriceTickSize, minQuantityTickSize, makerFeeRate, takerFeeRate sdk.Dec) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := types.NewSpotMarketLaunchProposal(title, description, ticker, baseDenom, quoteDenom, minPriceTickSize, minQuantityTickSize, &makerFeeRate, &takerFeeRate)
	return content, nil
}

func perpetualMarketLaunchArgsToContent(cmd *cobra.Command, ticker, quoteDenom, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType, initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize sdk.Dec) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := types.NewPerpetualMarketLaunchProposal(title, description, ticker, quoteDenom, oracleBase, oracleQuote, oracleScaleFactor, oracleType, initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize)
	return content, nil
}

func expiryFuturesMarketLaunchArgsToContent(cmd *cobra.Command, ticker, quoteDenom, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType, expiry int64, initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize sdk.Dec) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := types.NewExpiryFuturesMarketLaunchProposal(title, description, ticker, quoteDenom, oracleBase, oracleQuote, oracleScaleFactor, oracleType, expiry, initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize)
	return content, nil
}

func derivativeMarketParamUpdateArgsToContent(
	cmd *cobra.Command, marketID string,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize *sdk.Dec,
	hourlyInterestRate, hourlyFundingRateCap *sdk.Dec,
	oracleParams *types.OracleParams, status types.MarketStatus,
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

func buildExchangeAuthz(subaccountId string, marketIds []string, msgType string) authz.Authorization {
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

func buildBatchUpdateExchangeAuthz(subaccountId string, spotMarketIds, derivativeMarketIds []string) authz.Authorization {
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
