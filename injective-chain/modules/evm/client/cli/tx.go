package cli

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	rpctypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(NewRawTxCmd())
	return cmd
}

// NewRawTxCmd command build cosmos transaction from raw ethereum transaction
func NewRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw TX_HEX,TX_HEX_2,...",
		Short: "Build cosmos transaction from raw ethereum transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			rsp, err := rpctypes.NewQueryClient(clientCtx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			eip155ChainID := rsp.Params.ChainConfig.EIP155ChainID
			if eip155ChainID == nil {
				return errors.New("EIP155 ChainID is nil")
			}

			txBuilder := clientCtx.TxConfig.NewTxBuilder()
			totalFees := big.NewInt(0)
			msgs := make([]sdk.Msg, 0)
			totalGas := uint64(0)

			dataStrings := strings.Split(args[0], ",")
			for i, dataStr := range dataStrings {
				data, err := hexutil.Decode(dataStr)
				if err != nil {
					return errors.Wrapf(err, "failed to decode ethereum tx hex bytes #%d", i)
				}

				msg := &types.MsgEthereumTx{}
				if err := msg.UnmarshalBinary(data, ethtypes.LatestSignerForChainID(eip155ChainID.BigInt())); err != nil {
					return err
				}

				if err := msg.ValidateBasic(); err != nil {
					return err
				}

				totalFees.Add(totalFees, msg.GetFee())
				totalGas += msg.GetGas()

				// we ignore returned tx since this call is only to modify txBuilder state
				tx, err := msg.BuildTx(txBuilder, rsp.Params.EvmDenom)
				if err != nil {
					return err
				}

				msgs = append(msgs, tx.GetMsgs()...)
			}
			feeAmt := sdk.NewCoins(sdk.NewCoin(rsp.Params.EvmDenom, sdkmath.NewIntFromBigInt(totalFees)))
			txBuilder.SetFeeAmount(feeAmt)
			txBuilder.SetGasLimit(totalGas)
			txBuilder.SetMsgs(msgs...)
			tx := txBuilder.GetTx()

			if clientCtx.GenerateOnly {
				json, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
				if err != nil {
					return err
				}

				return clientCtx.PrintString(fmt.Sprintf("%s\n", json))
			}

			if !clientCtx.SkipConfirm {
				out, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
				if err != nil {
					return err
				}

				_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", out)

				buf := bufio.NewReader(os.Stdin)
				ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf, os.Stderr)

				if err != nil || !ok {
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", "canceled transaction")
					return err
				}
			}

			txBytes, err := clientCtx.TxConfig.TxEncoder()(tx)
			if err != nil {
				return err
			}

			// broadcast to a Tendermint node
			res, err := clientCtx.BroadcastTx(txBytes)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
