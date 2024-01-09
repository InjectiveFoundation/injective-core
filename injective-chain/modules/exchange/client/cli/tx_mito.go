package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

type Slippage struct {
	MaxPenalty   *sdk.Dec `json:"max_penalty,omitempty"`
	MinIncentive *sdk.Dec `json:"min_incentive,omitempty"`
}

type VaultSubscribe struct {
	Slippage *Slippage `json:"slippage,omitempty"`
}

type BasicVaultRedeemArgs struct {
	LpTokenBurnAmount math.Int `json:"lp_token_burn_amount"`
	Slippage          Slippage `json:"slippage,omitempty"`
}

type VaultRedeem struct {
	BasicVaultRedeemArgs
	RedemptionType string `json:"redemption_type,omitempty"`
}

type VaultSubscribeRedeem struct {
	Subscribe *VaultSubscribe `json:"subscribe,omitempty"`
	Redeem    interface{}     `json:"redeem,omitempty"`
}

type VaultInput struct {
	VaultSubaccountId  string               `json:"vault_subaccount_id"`
	TraderSubaccountId string               `json:"trader_subaccount_id"`
	Msg                VaultSubscribeRedeem `json:"msg"`
}

const MIN_INCENTIVE_DISABLED_FLAG = 101
const MAX_PENALTY_DEFAULT_VALUE = 1

func NewSubscribeToSpotVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe-to-spot-vault [vault subaccount id] [subscriber subaccount id] [flags]",
		Args:  cobra.MinimumNArgs(3),
		Short: "subscribe-to-spot-vault",
		Long: `subscribe-to-spot-vault.

		Example:
		$ %s tx exchange subscribe-to-spot-vault [vault address] [vault subaccount id] [subscriber subaccount id] (--quote-amount=<quote_amount>) (--base-amount=<base_amount>)  --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultAddress := args[0]
			vaultSubaccountId := args[1]
			traderSubaccountId := args[2]

			quoteAmountFlag, quoteErr := cmd.Flags().GetString(FlagSubscriptionQuoteAmount)

			if quoteErr != nil {
				return quoteErr
			}

			baseAmountFlag, baseErr := cmd.Flags().GetString(FlagSubscriptionBaseAmount)

			if baseErr != nil {
				return baseErr
			}

			var quoteAmount, baseAmount math.Int
			var ok bool

			if quoteAmountFlag == "" && baseAmountFlag == "" {
				return fmt.Errorf(
					"expected either quote or base amount or both, got but got neither",
				)
			}

			if quoteAmountFlag != "" {
				quoteAmount, ok = math.NewIntFromString(quoteAmountFlag)
				if !ok {
					return fmt.Errorf("failed to convert quote amount to int: %v", quoteAmountFlag)
				}
			}

			if baseAmountFlag != "" {
				baseAmount, ok = math.NewIntFromString(baseAmountFlag)
				if !ok {
					return fmt.Errorf("failed to convert base amount to int: %v", baseAmountFlag)
				}
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryMarketIDFromVaultRequest{
				VaultAddress: vaultAddress,
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
				bankFunds = append(bankFunds, sdk.NewCoin(quoteDenom, quoteAmount))
			}

			if baseAmountFlag != "" {
				bankFunds = append(bankFunds, sdk.NewCoin(baseDenom, baseAmount))
			}

			fromAddress := clientCtx.GetFromAddress().String()

			vaultSubscribe := VaultSubscribe{
				Slippage: nil,
			}

			forwardMsg := VaultSubscribeRedeem{
				Subscribe: &vaultSubscribe,
			}

			vaultInput := VaultInput{
				Msg:                forwardMsg,
				VaultSubaccountId:  vaultSubaccountId,
				TraderSubaccountId: traderSubaccountId,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultSubscribe",
				Args:   vaultInput,
			}

			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				return err
			}

			masterAddress := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           bankFunds.String(),
				ContractAddress: masterAddress.String(),
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

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSubscribeToAmmVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe-to-amm-vault [vault address] [vault subaccount id] [trader subaccount id] [flags]",
		Args:  cobra.MinimumNArgs(3),
		Short: "subscribe-to-amm-vault",
		Long: `subscribe-to-amm-vault.

		Example:
		$ %s tx exchange subscribe-to-amm-vault [vault address] [vault subaccount id] [subscriber subaccount id] (--quote-amount=<quote_amount>) (--base-amount=<base_amount>) (--max-penalty:=<max penalty:1>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultAddress := args[0]
			vaultSubaccountId := args[1]
			traderSubaccountId := args[2]

			quoteAmountFlag, quoteErr := cmd.Flags().GetString(FlagSubscriptionQuoteAmount)

			if quoteErr != nil {
				return quoteErr
			}

			baseAmountFlag, baseErr := cmd.Flags().GetString(FlagSubscriptionBaseAmount)

			if baseErr != nil {
				return baseErr
			}

			var quoteAmount, baseAmount math.Int
			var ok bool

			if quoteAmountFlag == "" && baseAmountFlag == "" {
				return fmt.Errorf(
					"expected either quote or base amount or both, got but got neither",
				)
			}

			if quoteAmountFlag != "" {
				quoteAmount, ok = math.NewIntFromString(quoteAmountFlag)
				if !ok {
					return fmt.Errorf("failed to convert quote amount to int: %v", quoteAmountFlag)
				}
			}

			if baseAmountFlag != "" {
				baseAmount, ok = math.NewIntFromString(baseAmountFlag)
				if !ok {
					return fmt.Errorf("failed to convert base amount to int: %v", baseAmountFlag)
				}
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryMarketIDFromVaultRequest{
				VaultAddress: vaultAddress,
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
				bankFunds = append(bankFunds, sdk.NewCoin(quoteDenom, quoteAmount))
			}

			if baseAmountFlag != "" {
				bankFunds = append(bankFunds, sdk.NewCoin(baseDenom, baseAmount))
			}

			fromAddress := clientCtx.GetFromAddress().String()

			slippage, slippageErr := getSlippage(cmd)
			if slippageErr != nil {
				return slippageErr
			}

			vaultSubscribe := VaultSubscribe{
				Slippage: &slippage,
			}

			forwardMsg := VaultSubscribeRedeem{
				Subscribe: &vaultSubscribe,
			}

			vaultInput := VaultInput{
				Msg:                forwardMsg,
				VaultSubaccountId:  vaultSubaccountId,
				TraderSubaccountId: traderSubaccountId,
			}

			execData := wasmxtypes.ExecutionData{
				Origin: fromAddress,
				Name:   "VaultSubscribe",
				Args:   vaultInput,
			}
			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				return err
			}

			masterAddress := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           bankFunds.String(),
				ContractAddress: masterAddress.String(),
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
	cmd.Flags().
		Int64(FlagSubscriptionMaxPenalty, MAX_PENALTY_DEFAULT_VALUE, "max penalty % to accept when redeeming only with single side <0,100>; 1 by default")

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
		$ %s tx exchange redeem-from-spot-vault [vault subaccount id] [trader subaccount id] [lp token burn amount] (--max-penalty:=<max penalty:1>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			traderSubaccountId := args[1]
			lpTokenBurnAmount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()
			slippage, slippageErr := getSlippage(cmd)

			if slippageErr != nil {
				return slippageErr
			}

			vaultRedeem := VaultRedeem{
				BasicVaultRedeemArgs: BasicVaultRedeemArgs{
					LpTokenBurnAmount: lpTokenBurnAmount.Amount,
					Slippage:          slippage,
				},
				RedemptionType: "FixedBaseAndQuote",
			}

			vaultInput := VaultInput{
				TraderSubaccountId: traderSubaccountId,
				VaultSubaccountId:  vaultSubaccountId,
				Msg: VaultSubscribeRedeem{
					Redeem: &vaultRedeem,
				},
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
				Funds:           lpTokenBurnAmount.String(),
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		Int64(FlagSubscriptionMaxPenalty, MAX_PENALTY_DEFAULT_VALUE, "max penalty % to accept when redeeming only with single side <0,100>; 1 by default")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRedeemFromAmmVaultTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redeem-from-amm-vault [flags]",
		Args:  cobra.MinimumNArgs(3),
		Short: "redeem-from-amm-vault",
		Long: `redeem-from-amm-vault.

		Example:
		$ %s tx exchange redeem-from-amm-vault [vault subaccount id] [trader subaccount id] [lp token burn amount] (--max-penalty:=<max penalty:1>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			traderSubaccountId := args[1]
			lpTokenBurnAmount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()
			slippage, slippageErr := getSlippage(cmd)

			if slippageErr != nil {
				return slippageErr
			}

			vaultRedeem := VaultRedeem{
				BasicVaultRedeemArgs: BasicVaultRedeemArgs{
					LpTokenBurnAmount: lpTokenBurnAmount.Amount,
					Slippage:          slippage,
				},
				RedemptionType: "FixedBaseAndQuote",
			}

			vaultInput := VaultInput{
				TraderSubaccountId: traderSubaccountId,
				VaultSubaccountId:  vaultSubaccountId,
				Msg: VaultSubscribeRedeem{
					Redeem: &vaultRedeem,
				},
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
				Funds:           lpTokenBurnAmount.String(),
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		Int64(FlagSubscriptionMaxPenalty, MAX_PENALTY_DEFAULT_VALUE, "max penalty % to accept when redeeming only with single side <0,100>; 1 by default")
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
		$ %s tx exchange subscribe-to-derivative-vault [vault address] [vault subaccount id] [subscriber subaccount id] [amount] (--max-penalty:=<max penalty:1>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultAddress := args[0]
			vaultSubaccountId := args[1]
			traderSubaccountId := args[2]

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryMarketIDFromVaultRequest{
				VaultAddress: vaultAddress,
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

			amount, ok := math.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid integer: %s", args[3])
			}

			bankFunds := sdk.NewCoins(sdk.NewCoin(quoteDenom, amount))
			fromAddress := clientCtx.GetFromAddress().String()

			slippage, slippageErr := getSlippage(cmd)
			if slippageErr != nil {
				return slippageErr
			}

			vaultSubscribe := VaultSubscribe{
				Slippage: &slippage,
			}

			forwardMsg := VaultSubscribeRedeem{
				Subscribe: &vaultSubscribe,
			}

			vaultInput := VaultInput{
				Msg:                forwardMsg,
				VaultSubaccountId:  vaultSubaccountId,
				TraderSubaccountId: traderSubaccountId,
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

			masterAddress := types.SubaccountIDToSdkAddress(common.HexToHash(vaultSubaccountId))
			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           bankFunds.String(),
				ContractAddress: masterAddress.String(),
				Data:            string(execDataBytes),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		Int64(FlagSubscriptionMaxPenalty, MAX_PENALTY_DEFAULT_VALUE, "max penalty % to accept when subscribing without position <0,100>; 1 by default")
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
		$ %s tx exchange redeem-from-derivative-vault [vault subaccount id] [trader subaccount id] [lp token burn amount] (redemption type) (--max-penalty:=<max penalty:1>) --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaultSubaccountId := args[0]
			traderSubaccountId := args[1]
			lpTokenBurnAmount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()

			redemptionType := "PositionAndQuote"
			if len(args) == 4 {
				switch args[3] {
				case "QuoteOnly":
					redemptionType = "QuoteOnly"
				case "PositionAndQuote":
				default:
					return fmt.Errorf(
						"invalid redemption type. Only 'QuoteOnly' and 'PositionAndQuote' are supported, but '%s' was given",
						args[3],
					)
				}
			}

			slippage, slippageErr := getSlippage(cmd)

			if slippageErr != nil {
				return slippageErr
			}

			vaultRedeem := VaultRedeem{
				BasicVaultRedeemArgs: BasicVaultRedeemArgs{
					LpTokenBurnAmount: lpTokenBurnAmount.Amount,
					Slippage:          slippage,
				},
				RedemptionType: redemptionType,
			}

			vaultInput := VaultInput{
				VaultSubaccountId:  vaultSubaccountId,
				TraderSubaccountId: traderSubaccountId,
				Msg: VaultSubscribeRedeem{
					Redeem: &vaultRedeem,
				},
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
				Funds:           lpTokenBurnAmount.String(),
				ContractAddress: vaultAccount.String(),
				Data:            string(execDataBytes),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		Int64(FlagSubscriptionMaxPenalty, MAX_PENALTY_DEFAULT_VALUE, "max penalty % to accept when redeeming without position <0,100>; 1 by default")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewPrivilegedExecuteContractTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privileged-execute-contract [contract address] [wasmx execution data message] --from=genesis --keyring-backend=file --yes",
		Args:  cobra.ExactArgs(2),
		Short: "Executes any smart contract message with privileged actions for the exchange module as part of the contract's response data.",
		Long: `Executes any smart contract message with privileged actions for the exchange module as part of the contract's response data.

		Example:
		$ %s tx exchange privileged-execute-contract inj1zlwdkv49rmsug0pnwu6fmwnl267lfr34yvhwgp '{"origin":"inj17gkuet8f6pssxd8nycm3qr9d9y699rupv6397z", "name":"Subscribe", "args": {"Subscribe":{"args": {"subscriber_subaccount_id":"0xf22dccace9d0610334f32637100cad2934528f81000000000000000000000000"}}}}' --from=wasm --funds=1000000000000000000inj,10000000peggy0xdAC17F958D2ee523a2206206994597C13D831ec7
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddress := args[0]
			sdk.MustAccAddressFromBech32(contractAddress)
			rawMessage := args[1]

			fromAddress := clientCtx.GetFromAddress().String()

			fundsStr, err := cmd.Flags().GetString(FlagFunds)
			if err != nil {
				return err
			}

			var amountStr string

			if fundsStr == "" {
				amountStr = ""
			} else {

				amount, err := sdk.ParseCoinsNormalized(fundsStr)
				if err != nil {
					return err
				}

				amountStr = amount.String()
			}

			asMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(rawMessage), &asMap)
			if err != nil {
				return err
			}

			execData := wasmxtypes.ExecutionData{
				Origin: asMap["origin"].(string),
				Name:   asMap["name"].(string),
				Args:   asMap["args"],
			}

			var execDataBytes []byte
			execDataBytes, err = json.Marshal(execData)
			if err != nil {
				return err
			}

			msg := &types.MsgPrivilegedExecuteContract{
				Sender:          fromAddress,
				Funds:           amountStr,
				ContractAddress: contractAddress,
				Data:            string(execDataBytes),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().
		String(FlagFunds, "", "funds to pass to the contract")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getSlippage(cmd *cobra.Command) (Slippage, error) {
	maxPenaltyFlag, maxPenaltyErr := cmd.Flags().GetInt64(FlagSubscriptionMaxPenalty)
	if maxPenaltyErr != nil {
		return Slippage{}, maxPenaltyErr
	}

	if maxPenaltyFlag < 0 || maxPenaltyFlag > 100 {
		return Slippage{}, fmt.Errorf(
			"max penalty has to be within <0,100>, but %d was given",
			maxPenaltyFlag,
		)
	}

	penaltyDec := sdk.NewDecFromInt(math.NewInt(maxPenaltyFlag))
	slippage := Slippage{
		MaxPenalty: &penaltyDec,
	}

	return slippage, nil
}
