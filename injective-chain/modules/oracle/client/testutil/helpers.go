//nolint:staticcheck // deprecated gov proposal flags
package testutil

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtestutil "github.com/cosmos/cosmos-sdk/x/gov/client/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/client/cli"
)

// commonArgs is args for CLI test commands
var commonArgs = []string{
	fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
	fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	fmt.Sprintf("--%s=%s", flags.FlagChainID, "injective-1"),
}

func GrantPriceFeederPrivilege(net *network.Network, clientCtx client.Context, base, quote, relayers string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		base, quote, relayers,
		fmt.Sprintf("--%s=%s", govcli.FlagTitle, "grant price feeder privilege proposal"),
		fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Where is the title!?"),
		fmt.Sprintf("--%s=%s", govcli.FlagDeposit, sdk.NewCoin(sdk.DefaultBondDenom, govtypes.DefaultMinDepositTokens).String()),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewGrantPriceFeederPrivilegeProposalTxCmd()
	output, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
	if err != nil {
		return output, err
	}
	txResp := sdk.TxResponse{}
	err = clientCtx.Codec.UnmarshalJSON(output.Bytes(), &txResp)
	if err != nil {
		return output, err
	}
	txResp, _ = clitestutil.GetTxResponse(net, clientCtx, txResp.TxHash)
	if len(txResp.Logs) == 0 {
		return output, errors.New("proposal log does not exist")
	}

	if txResp.Logs[0].Events[1].Attributes[0].Key != "proposal_id" {
		return output, errors.New("proposal_id event is not set in correct place")
	}

	proposalID := txResp.Logs[0].Events[1].Attributes[0].Value

	// vote
	return govtestutil.MsgVote(clientCtx, from.String(), proposalID, "yes")
}

func GrantBandOraclePrivilege(net *network.Network, clientCtx client.Context, relayers string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		relayers,
		fmt.Sprintf("--%s=%s", govcli.FlagTitle, "grant price feeder privilege proposal"),
		fmt.Sprintf("--%s=%s", govcli.FlagDescription, "Where is the title!?"),
		fmt.Sprintf("--%s=%s", govcli.FlagDeposit, sdk.NewCoin(sdk.DefaultBondDenom, govtypes.DefaultMinDepositTokens).String()),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10))).String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewGrantBandOraclePrivilegeProposalTxCmd()
	output, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
	if err != nil {
		return output, err
	}

	txResp := sdk.TxResponse{}
	err = clientCtx.Codec.UnmarshalJSON(output.Bytes(), &txResp)
	if err != nil {
		return output, err
	}
	if len(txResp.Logs) == 0 {
		return output, errors.New("proposal log does not exist")
	}

	if txResp.Logs[0].Events[4].Attributes[0].Key != "proposal_id" {
		return output, errors.New("proposal_id event is not set in correct place")
	}

	proposalID := txResp.Logs[0].Events[4].Attributes[0].Value
	output, err = govtestutil.MsgVote(clientCtx, from.String(), proposalID, "yes")
	return output, err
}

// TODO: add helper for below proposals if required
// - NewRevokePriceFeederPrivilegeProposalTxCmd
// - NewRevokeBandOraclePrivilegeProposalTxCmd

func MsgRelayPriceFeedPrice(net *network.Network, clientCtx client.Context, base, quote, price string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		base, quote, price,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewRelayPriceFeedPriceTxCmd()

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
		return output, fmt.Errorf("tx response code is not 0 during MsgRelayPriceFeedPrice: %d, log: %s", txResp.Code, txResp.RawLog)
	}
	txResp, err = clitestutil.GetTxResponse(net, clientCtx, txResp.TxHash)
	if err != nil {
		return output, err
	}
	if txResp.Code != uint32(0) {
		return output, fmt.Errorf("tx response code is not 0 during MsgRelayPriceFeedPrice: %d, log: %s", txResp.Code, txResp.RawLog)
	}

	return output, err
}

func MsgRelayBandRates(clientCtx client.Context, symbols, rates, resolveTimes, requestIDs string, from fmt.Stringer, extraArgs ...string) (testutil.BufferWriter, error) {
	args := []string{
		symbols, rates, resolveTimes, requestIDs,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, from.String()),
	}

	args = append(args, commonArgs...)

	cmd := cli.NewRelayBandRatesTxCmd()

	return clitestutil.ExecTestCLICmd(clientCtx, cmd, append(args, extraArgs...))
}
