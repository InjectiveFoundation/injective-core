package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/client/cli"
)

// commonArgs is args for CLI test commands
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	fmt.Sprintf("--%s=%s", flags.FlagChainID, "injective-1"),
}

func MsgCreateInsuranceFund(net *network.Network, clientCtx client.Context, ticker, quoteDenom, oracleBase, oracleQuote, oracleType, expiry, initialDeposit string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%s", cli.FlagTicker, ticker),
		fmt.Sprintf("--%s=%s", cli.FlagQuoteDenom, quoteDenom),
		fmt.Sprintf("--%s=%s", cli.FlagOracleBase, oracleBase),
		fmt.Sprintf("--%s=%s", cli.FlagOracleQuote, oracleQuote),
		fmt.Sprintf("--%s=%s", cli.FlagOracleType, oracleType),
		fmt.Sprintf("--%s=%s", cli.FlagExpiry, expiry),
		fmt.Sprintf("--%s=%s", cli.FlagInitialDeposit, initialDeposit),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewCreateInsuranceFundTxCmd()

	output, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
	if err != nil {
		return output, err
	}
	txResp := sdk.TxResponse{}
	err = clientCtx.Codec.UnmarshalJSON(output.Bytes(), &txResp)
	if err != nil {
		return output, err
	}
	if txResp.Code != uint32(0) {
		return output, fmt.Errorf("tx response code is not 0 during MsgCreateInsuranceFund: %d, log: %s", txResp.Code, txResp.RawLog)
	}
	txResp, err = clitestutil.GetTxResponse(net, clientCtx, txResp.TxHash)
	if err != nil {
		return output, err
	}
	if txResp.Code != uint32(0) {
		return output, fmt.Errorf("tx response code is not 0 during MsgCreateInsuranceFund: %d, log: %s", txResp.Code, txResp.RawLog)
	}

	return output, err
}
