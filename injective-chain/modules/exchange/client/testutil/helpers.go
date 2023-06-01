package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/client/cli"
)

// commonArgs is args for CLI test commands
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	fmt.Sprintf("--%s=%s", flags.FlagChainID, "injective-1"),
}

func MsgInstantSpotMarketLaunch(network *network.Network, clientCtx client.Context, ticker, baseDenom, quoteDenom string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		ticker, baseDenom, quoteDenom,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewInstantSpotMarketLaunchTxCmd()

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
		return output, fmt.Errorf("tx response code is not 0 during MsgInstantSpotMarketLaunch: %d, log: %s", txResp.Code, txResp.RawLog)
	}
	txResp, err = clitestutil.GetTxResponse(network, clientCtx, txResp.TxHash)
	if err != nil {
		return output, err
	}
	if txResp.Code != uint32(0) {
		return output, fmt.Errorf("tx response code is not 0 during MsgInstantSpotMarketLaunch: %d, log: %s", txResp.Code, txResp.RawLog)
	}

	return output, err
}

func MsgInstantPerpetualMarketLaunch(network *network.Network, clientCtx client.Context, ticker, quoteDenom, oracleBase, oracleQuote string, oracleTypeStr string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		fmt.Sprintf("--%s=%s", cli.FlagTicker, ticker),
		fmt.Sprintf("--%s=%s", cli.FlagQuoteDenom, quoteDenom),
		fmt.Sprintf("--%s=%s", cli.FlagOracleBase, oracleBase),
		fmt.Sprintf("--%s=%s", cli.FlagOracleQuote, oracleQuote),
		fmt.Sprintf("--%s=%s", cli.FlagOracleType, oracleTypeStr),
		fmt.Sprintf("--%s=%d", cli.FlagOracleScaleFactor, 0),
		fmt.Sprintf("--%s=%s", cli.FlagMakerFeeRate, "0.001"),
		fmt.Sprintf("--%s=%s", cli.FlagTakerFeeRate, "0.001"),
		fmt.Sprintf("--%s=%s", cli.FlagInitialMarginRatio, "0.05"),
		fmt.Sprintf("--%s=%s", cli.FlagMaintenanceMarginRatio, "0.02"),
		fmt.Sprintf("--%s=%s", cli.FlagMinPriceTickSize, "0.0001"),
		fmt.Sprintf("--%s=%s", cli.FlagMinQuantityTickSize, "0.001"),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewInstantPerpetualMarketLaunchTxCmd()

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
		return output, fmt.Errorf("tx response code is not 0 during MsgInstantPerpetualMarketLaunch: %d, log: %s", txResp.Code, txResp.RawLog)
	}
	txResp, err = clitestutil.GetTxResponse(network, clientCtx, txResp.TxHash)
	if err != nil {
		return output, err
	}
	if txResp.Code != uint32(0) {
		return output, fmt.Errorf("tx response code is not 0 during MsgInstantPerpetualMarketLaunch: %d, log: %s", txResp.Code, txResp.RawLog)
	}

	return output, err
}

func MsgInstantExpiryFuturesMarketLaunch(clientCtx client.Context, ticker, quoteDenom, oracleBase, oracleQuote string, oracleTypeStr string, expiry string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		ticker, quoteDenom, oracleBase, oracleQuote, oracleTypeStr, expiry,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewInstantExpiryFuturesMarketLaunchTxCmd()

	return clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
}
