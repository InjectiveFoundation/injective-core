package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetQueryCmd returns the parent command for all modules/bank CLi query commands.
func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		GetAllSpotMarkets(),
		GetAllDerivativeMarkets(),
		GetSpotMarket(),
		GetDerivativeMarket(),
		GetExchangeParamsCmd(),
		GetSubaccountDeposits(),
		GetSubaccountDeposit(),
		GetEthAddressFromInjAddressCmd(),
		GetInjAddressFromEthAddressCmd(),
		GetSubaccountIDFromInjAddressCmd(),
		GetAllBinaryOptionsMarketsCmd(),
	)
	return cmd
}

// GetAllSpotMarkets queries all active spot markets
func GetAllSpotMarkets() *cobra.Command {
	cmd := cli.QueryCmd("spot-markets",
		"Gets all active spot markets",
		types.NewQueryClient,
		&types.QuerySpotMarketsRequest{
			Status: "Active",
		}, cli.FlagsMapping{
			"MarketIds": cli.Flag{Flag: FlagMarketIDs},
		}, cli.ArgsMapping{})
	cmd.Long = "Gets all active spot markets. If the height is not provided, it will use the latest height from context."
	cmd.Flags().String(FlagMarketIDs, "", "filter by market IDs, comma-separated")
	return cmd
}

// GetSpotMarket queries a single spot market
func GetSpotMarket() *cobra.Command {
	cmd := cli.QueryCmd("spot-market <market_id>",
		"Gets a single spot market",
		types.NewQueryClient,
		&types.QuerySpotMarketRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{})
	cmd.Long = "Gets a single spot market by ID. If the height is not provided, it will use the latest height from context."
	return cmd
}

// GetSubaccountDeposits queries all subaccount deposits
func GetSubaccountDeposits() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposits [trader] [nonce]",
		Short: "Gets all the deposits of a given subaccount",
		Long:  "Gets all the deposits of a given subaccount. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			nonce, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			req := &types.QuerySubaccountDepositsRequest{
				Subaccount: &types.Subaccount{
					Trader:          args[0],
					SubaccountNonce: uint32(nonce),
				},
			}
			res, err := queryClient.SubaccountDeposits(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetSubaccountDeposit queries a subaccount's deposits for a given denomination
func GetSubaccountDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [subaccount_id] [denom]",
		Short: "Gets the deposits of a given subaccount for a given denomination",
		Long:  "Gets the deposits of a given subaccount. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QuerySubaccountDepositRequest{
				SubaccountId: args[0],
				Denom:        args[1],
			}
			res, err := queryClient.SubaccountDeposit(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetAllDerivativeMarkets queries all active derivative markets
func GetAllDerivativeMarkets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "derivative-markets",
		Short: "Gets all active derivative markets",
		Long:  "Gets all active derivative markets. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryDerivativeMarketsRequest{
				Status: "Active",
			}
			res, err := queryClient.DerivativeMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetDerivativeMarket queries a single derivative market
func GetDerivativeMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "derivative-market [market_id]",
		Short: "Gets a single derivative market by Market ID",
		Long:  "Gets a single derivative market by Market ID. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryDerivativeMarketRequest{
				MarketId: args[0],
			}
			res, err := queryClient.DerivativeMarket(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetExchangeParamsCmd queries exchange params info
func GetExchangeParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets exchange params info.",
		Long:  "Gets exchange params info.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryExchangeParamsRequest{}

			res, err := queryClient.QueryExchangeParams(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetEthAddressFromInjAddressCmd returns the Injective address for an account given its hex-encoded Ethereum address
func GetEthAddressFromInjAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eth-address-from-inj-address [Injective address]",
		Short: "Returns the Ethereum address given an inj-prefixed Cosmos address",
		Long:  "Returns the Ethereum address given an inj-prefixed Cosmos address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			address, err := AccountToHex(args[0])
			if err != nil {
				return err
			}
			return clientCtx.PrintString(address + "\n")
		},
	}
	return cmd
}

// GetSubaccountIDFromInjAddressCmd returns the default subaccount ID for an account given its INJ address
func GetSubaccountIDFromInjAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subaccount-id-from-inj-address [Injective address]",
		Short: "Returns the default Subaccount ID given an inj-prefixed Cosmos address",
		Long:  "Returns the default Subaccount ID given an inj-prefixed Cosmos address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			address, err := AccountToHex(args[0])
			if err != nil {
				return err
			}
			subaccountId := fmt.Sprintf("%s%024x", address, 0)
			return clientCtx.PrintString(subaccountId + "\n")
		},
	}
	return cmd
}

// GetInjAddressFromEthAddressCmd returns the Ethereum address for an account given its INJ address
func GetInjAddressFromEthAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inj-address-from-eth-address [Ethereum address]",
		Short: "Returns the INJ address given an hex-encoded Ethereum address",
		Long:  "Returns the INJ address given an hex-encoded Ethereum address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			address := common.HexToAddress(args[0])

			return clientCtx.PrintString(sdk.AccAddress(address.Bytes()).String() + "\n")
		},
	}
	return cmd
}

func AccountToHex(addr string) (string, error) {
	if strings.HasPrefix(addr, sdk.GetConfig().GetBech32AccountAddrPrefix()) {
		// Check to see if address is Cosmos bech32 formatted
		toAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return "", errors.Wrap(err, "must provide a valid Bech32 address")
		}
		ethAddr := common.BytesToAddress(toAddr.Bytes())
		return ethAddr.Hex(), nil
	}

	if !strings.HasPrefix(addr, "0x") {
		addr = "0x" + addr
	}

	valid := common.IsHexAddress(addr)
	if !valid {
		return "", fmt.Errorf("%s is not a valid Ethereum or Cosmos address", addr)
	}

	ethAddr := common.HexToAddress(addr)

	return ethAddr.Hex(), nil
}

// GetAllBinaryOptionsMarketsCmd queries all active binary option markets
func GetAllBinaryOptionsMarketsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "binary-options-markets",
		Short: "Gets all active binary options markets",
		Long:  "Gets all active binary options markets. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryBinaryMarketsRequest{
				Status: "Active",
			}
			res, err := queryClient.BinaryOptionsMarkets(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}
