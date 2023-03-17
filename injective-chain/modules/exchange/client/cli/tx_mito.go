package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type BasicVaultSubscribeArgs struct {
	VaultSubaccountId      string `json:"vault_subaccount_id"`
	SubscriberSubaccountId string `json:"subscriber_subaccount_id"`
}

type SpotVaultSubscribeArgs struct {
	BasicVaultSubscribeArgs
	SubscriptionType SpotSubscriptionType `json:"subscription_type,omitempty"`
	MaxSlippage      sdk.Dec              `json:"max_slippage,omitempty"`
	Deadline         uint64               `json:"deadline,omitempty"`
}

type DerivativeVaultSubscribeArgs struct {
	BasicVaultSubscribeArgs
	MarginRatio      *sdk.Dec                   `json:"margin_ratio,omitempty"`
	SubscriptionType DerivativeSubscriptionType `json:"subscription_type,omitempty"`
}

type SpotSubscriptionType struct {
	SpotSubscriptionType string `json:"SpotSubscriptionType"`
}

type DerivativeSubscriptionType struct {
	DerivativeSubscriptionType string `json:"DerivativeSubscriptionType"`
}

type VaultSubscribe struct {
	SubscribeArgs interface{} `json:"args"`
}

type DerivativeRedemptionType struct {
	DerivativeRedemptionType string `json:"DerivativeRedemptionType"`
}

type SpotRedemptionType struct {
	SpotRedemptionType string `json:"SpotRedemptionType"`
}

type BasicVaultRedeemArgs struct {
	VaultSubaccountId    string  `json:"vault_subaccount_id"`
	RedeemerSubaccountId string  `json:"redeemer_subaccount_id"`
	LpTokenBurnAmount    sdk.Int `json:"lp_token_burn_amount"`
}

type SpotVaultRedeemArgs struct {
	BasicVaultRedeemArgs
	RedemptionType  SpotRedemptionType `json:"redemption_type,omitempty"`
	RedemptionRatio *sdk.Dec           `json:"redemption_ratio,omitempty"`
}

type DerivativeVaultRedeemArgs struct {
	BasicVaultRedeemArgs
	RedemptionType DerivativeRedemptionType `json:"redemption_type,omitempty"`
}

type VaultRedeem struct {
	RedeemArgs interface{} `json:"args"`
}

type VaultInput struct {
	Subscribe *VaultSubscribe `json:"Subscribe,omitempty"`
	Redeem    *VaultRedeem    `json:"Redeem,omitempty"`
}

func NewSubscribeToSpotVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe-to-spot-vault [vault subaccount id] [subscriber subaccount id] [flags]",
		Args:  cobra.MinimumNArgs(2),
		Short: "subscribe-to-spot-vault",
		Long: `subscribe-to-spot-vault.

		Example:
		$ %s tx exchange subscribe-to-spot-vault [vault subaccount id] [subscriber subaccount id] (--quote-amount=<quote_amount>) (--base-amount=<base_amount>) (--max-slippage=<max slippage:-1>) (--deadline=<deadline:now + 10s>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			subscriberSubaccountId := args[1]

			quoteAmountFlag, quoteErr := cmd.Flags().GetString(FlagSubscriptionQuoteAmount)

			if quoteErr != nil {
				return quoteErr
			}

			baseAmountFlag, baseErr := cmd.Flags().GetString(FlagSubscriptionQuoteAmount)

			if baseErr != nil {
				return baseErr
			}

			var quoteAmount, baseAmount int64

			if quoteAmountFlag == "" && baseAmountFlag == "" {
				return fmt.Errorf("expected either quote or base amount or both, got but got neither")
			}

			if quoteAmountFlag != "" {
				q, err := strconv.Atoi(quoteAmountFlag)
				quoteAmount = int64(q)
				if err != nil {
					return err
				}
			}

			if baseAmountFlag != "" {
				b, err := strconv.Atoi(baseAmountFlag)
				baseAmount = int64(b)
				if err != nil {
					return err
				}
			}

			maxSlippageFlag, maxSlippageErr := cmd.Flags().GetInt64(FlagSubscriptionMaxSlippage)
			if maxSlippageErr != nil {
				return maxSlippageErr
			}

			if maxSlippageFlag < -100 || maxSlippageFlag > 100 {
				return fmt.Errorf("max slippage has to be within <-100,100>, but %d was given", maxSlippageFlag)
			}

			maxSlippage := sdk.NewDecFromInt(sdk.NewInt(maxSlippageFlag))

			deadlineFlag, deadlineErr := cmd.Flags().GetUint(FlagSubscriptionDeadline)
			if deadlineErr != nil {
				return deadlineErr
			}

			if deadlineFlag <= 0 {
				return errors.New("deadline must be > 0")
			}

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

			bankFunds := sdk.Coins{}

			if quoteAmountFlag != "" {
				bankFunds = append(bankFunds, sdk.NewCoin(quoteDenom, sdk.NewInt(quoteAmount)))
			}

			if baseAmountFlag != "" {
				bankFunds = append(bankFunds, sdk.NewCoin(baseDenom, sdk.NewInt(baseAmount)))
			}

			deadline := time.Now().UnixNano() + int64(deadlineFlag*1_000_000)

			fromAddress := clientCtx.GetFromAddress().String()
			spotSubscriptionType := SpotSubscriptionType{SpotSubscriptionType: "Regular"}

			vaultSubscribeArgs := SpotVaultSubscribeArgs{
				BasicVaultSubscribeArgs: BasicVaultSubscribeArgs{
					VaultSubaccountId:      vaultSubaccountId,
					SubscriberSubaccountId: subscriberSubaccountId,
				},
				SubscriptionType: spotSubscriptionType,
				MaxSlippage:      maxSlippage,
				Deadline:         uint64(deadline),
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

	cmd.Flags().String(FlagSubscriptionQuoteAmount, "", "quote amount to subscribe with")
	cmd.Flags().String(FlagSubscriptionBaseAmount, "", "base amount to subscribe with")
	cmd.Flags().Int64(FlagSubscriptionMaxSlippage, int64(-1), "max penalty % to accept when subscribing only with single side <-100,100>; -1 by default")
	cmd.Flags().Uint(FlagSubscriptionDeadline, 10, "subscription deadline from now in seconds; 10s by default")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRedeemFromSpotVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-from-spot-vault [flags]",
		Args:  cobra.MinimumNArgs(3),
		Short: "redeem-from-spot-vault",
		Long: `redeem-from-spot-vault.

		Example:
		$ %s tx exchange redeem-from-spot-vault [vault subaccount id] [redeemer subaccount id] [lp token burn amount] (redemption type) --from=genesis --keyring-backend=file --yes
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

			redemptionType := SpotRedemptionType{SpotRedemptionType: "BaseAndQuote"}
			if len(args) == 4 {
				switch args[3] {
				case "BaseOnly", "QuoteOnly", "FixedBaseAndQuote", "VariableBaseAndQuote":
					redemptionType = SpotRedemptionType{SpotRedemptionType: args[3]}
				default:
					return fmt.Errorf("invalid redemption type. Only 'QuoteOnly', 'PositionOnly', 'VariableBaseAndQuote' and 'FixedBaseAndQuote' are supported, but '%s' was given", args[3])
				}
			}

			vaultRedeemArgs := SpotVaultRedeemArgs{
				BasicVaultRedeemArgs: BasicVaultRedeemArgs{
					LpTokenBurnAmount:    lpTokenBurnAmount,
					VaultSubaccountId:    vaultSubaccountId,
					RedeemerSubaccountId: redeemerSubaccountId,
				},
				RedemptionType: redemptionType,
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
		Args:  cobra.MinimumNArgs(4),
		Short: "subscribe-to-derivative-vault",
		Long: `subscribe-to-derivative-vault.

		Example:
		$ %s tx exchange subscribe-to-derivative-vault [vault subaccount id] [subscriber subaccount id] [margin ratio] [amount] (subscription type) --from=genesis --keyring-backend=file --yes
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

			derivativeSubscriptionType := DerivativeSubscriptionType{DerivativeSubscriptionType: "WithoutPosition"}
			if len(args) == 5 {
				switch args[4] {
				case "WithPosition":
					derivativeSubscriptionType = DerivativeSubscriptionType{DerivativeSubscriptionType: args[4]}
				case "WithoutPosition":
				default:
					return fmt.Errorf("invalid subscription type. Only 'WithPosition' and 'WithoutPosition' is supported, but '%s' was given", args[3])
				}
			}

			vaultSubscribeArgs := DerivativeVaultSubscribeArgs{
				BasicVaultSubscribeArgs: BasicVaultSubscribeArgs{
					VaultSubaccountId:      vaultSubaccountId,
					SubscriberSubaccountId: subscriberSubaccountId,
				},
				MarginRatio:      &subscriberMarginRatio,
				SubscriptionType: derivativeSubscriptionType,
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
		Args:  cobra.MinimumNArgs(3),
		Short: "redeem-from-derivative-vault",
		Long: `redeem-from-derivative-vault.

		Example:
		$ %s tx exchange redeem-from-derivative-vault [vault subaccount id] [redeemer subaccount id] [lp token burn amount] (redemption type) --from=genesis --keyring-backend=file --yes
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

			redemptionType := DerivativeRedemptionType{DerivativeRedemptionType: "PositionAndQuote"}
			if len(args) == 4 {
				switch args[3] {
				case "QuoteOnly", "PositionOnly":
					redemptionType = DerivativeRedemptionType{DerivativeRedemptionType: args[3]}
				case "PositionAndQuote":
				default:
					return fmt.Errorf("invalid redemption type. Only 'QuoteOnly', 'PositionOnly' and 'PositionAndQuote' are supported, but '%s' was given", args[3])
				}
			}

			vaultRedeemArgs := DerivativeVaultRedeemArgs{
				BasicVaultRedeemArgs: BasicVaultRedeemArgs{
					LpTokenBurnAmount:    lpTokenBurnAmount,
					VaultSubaccountId:    vaultSubaccountId,
					RedeemerSubaccountId: redeemerSubaccountId,
				},
				RedemptionType: redemptionType,
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
