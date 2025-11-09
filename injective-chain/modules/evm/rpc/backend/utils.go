package backend

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	crypto "github.com/cometbft/cometbft/api/cometbft/crypto/v1"
	cmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/rpc/types"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

type txGasAndReward struct {
	gasUsed uint64
	reward  *big.Int
}

type sortGasAndReward []txGasAndReward

func (s sortGasAndReward) Len() int { return len(s) }
func (s sortGasAndReward) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortGasAndReward) Less(i, j int) bool {
	return s[i].reward.Cmp(s[j].reward) < 0
}

// getAccountNonce returns the account nonce for the given account address.
// If the pending value is true, it will iterate over the mempool (pending)
// txs in order to compute and return the pending tx sequence.
// Todo: include the ability to specify a blockNumber
func (b *Backend) getAccountNonce(accAddr common.Address, pending bool, height int64, logger log.Logger) (uint64, error) {
	queryClient := authtypes.NewQueryClient(b.clientCtx)
	adr := sdk.AccAddress(accAddr.Bytes()).String()
	ctx := types.ContextWithHeight(height)
	res, err := queryClient.Account(ctx, &authtypes.QueryAccountRequest{Address: adr})
	if err != nil {
		st, ok := status.FromError(err)
		// treat as account doesn't exist yet
		if ok && st.Code() == codes.NotFound {
			return 0, nil
		}
		return 0, err
	}
	var acc sdk.AccountI
	if err := b.clientCtx.InterfaceRegistry.UnpackAny(res.Account, &acc); err != nil {
		return 0, err
	}

	nonce := acc.GetSequence()

	if !pending {
		return nonce, nil
	}

	// the account retriever doesn't include the uncommitted transactions on the nonce so we need to
	// to manually add them.
	pendingTxs, err := b.PendingTransactions()
	if err != nil {
		logger.Error("failed to fetch pending transactions", "error", err.Error())
		return nonce, nil
	}

	// add the uncommitted txs to the nonce counter
	// only supports `MsgEthereumTx` style tx
	for _, tx := range pendingTxs {
		for _, msg := range tx.GetMsgs() {
			ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				// not ethereum tx
				break
			}

			sender, err := ethMsg.GetSenderLegacy(ethtypes.LatestSignerForChainID(b.ChainID().ToInt()))
			if err != nil {
				continue
			}
			if sender == accAddr {
				nonce++
			}
		}
	}

	return nonce, nil
}

// output: targetOneFeeHistory
func (b *Backend) processBlock(
	cometBlock *cmrpctypes.ResultBlock,
	ethBlock map[string]interface{},
	rewardPercentiles []float64,
	cometBlockResult *cmrpctypes.ResultBlockResults,
	targetOneFeeHistory *types.OneFeeHistory,
) error {
	blockHeight := cometBlock.Block.Height
	blockBaseFee, err := b.BaseFee(cometBlockResult)
	if err != nil || blockBaseFee == nil {
		targetOneFeeHistory.BaseFee = big.NewInt(0)
	} else {
		targetOneFeeHistory.BaseFee = blockBaseFee
	}

	// set gas used ratio
	gasLimitUint64, ok := ethBlock["gasLimit"].(hexutil.Uint64)
	if !ok {
		return fmt.Errorf("invalid gas limit type: %T", ethBlock["gasLimit"])
	}

	gasUsedBig, ok := ethBlock["gasUsed"].(*hexutil.Big)
	if !ok {
		return fmt.Errorf("invalid gas used type: %T", ethBlock["gasUsed"])
	}

	targetOneFeeHistory.NextBaseFee = new(big.Int)

	gasusedfloat, _ := new(big.Float).SetInt(gasUsedBig.ToInt()).Float64()

	if gasLimitUint64 <= 0 {
		return fmt.Errorf("gasLimit of block height %d should be bigger than 0 , current gaslimit %d", blockHeight, gasLimitUint64)
	}

	gasUsedRatio := gasusedfloat / float64(gasLimitUint64)
	blockGasUsed := gasusedfloat
	targetOneFeeHistory.GasUsedRatio = gasUsedRatio

	rewardCount := len(rewardPercentiles)
	targetOneFeeHistory.Reward = make([]*big.Int, rewardCount)
	for i := 0; i < rewardCount; i++ {
		targetOneFeeHistory.Reward[i] = big.NewInt(0)
	}

	// check cometTxs
	cometTxs := cometBlock.Block.Txs
	cometTxResults := cometBlockResult.TxResults
	cometTxCount := len(cometTxs)

	var sorter sortGasAndReward

	for i := 0; i < cometTxCount; i++ {
		eachcometTx := cometTxs[i]
		eachcometTxResult := cometTxResults[i]

		tx, err := b.clientCtx.TxConfig.TxDecoder()(eachcometTx)
		if err != nil {
			b.logger.Debug("failed to decode transaction in block", "height", blockHeight, "error", err.Error())
			continue
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			b.logger.Debug("transaction in a block is not FeeTx?", "height", blockHeight, "tx_index", i)
			continue
		}
		txGasUsed := uint64(eachcometTxResult.GasUsed)
		feeDenom := feeTx.GetFee().GetDenomByIndex(0)
		fee := feeTx.GetFee().AmountOf(feeDenom)
		gas := feeTx.GetGas()
		gasPrice := fee.Quo(math.NewIntFromUint64(gas))
		reward := gasPrice.Sub(math.NewIntFromBigInt(blockBaseFee)).BigInt()
		if reward.Sign() < 0 { // can be the case if tx is already in the mempool but BaseFee for the next block is already higher than tx's gas price
			reward = big.NewInt(0)
		}

		sorter = append(sorter, txGasAndReward{gasUsed: txGasUsed, reward: reward})
	}

	// return an all zero row if there are no transactions to gather data from
	ethTxCount := len(sorter)
	if ethTxCount == 0 {
		return nil
	}

	sort.Sort(sorter)

	var txIndex int
	sumGasUsed := sorter[0].gasUsed

	for i, p := range rewardPercentiles {
		thresholdGasUsed := uint64(blockGasUsed * p / 100)
		for sumGasUsed < thresholdGasUsed && txIndex < ethTxCount-1 {
			txIndex++
			sumGasUsed += sorter[txIndex].gasUsed
		}
		targetOneFeeHistory.Reward[i] = sorter[txIndex].reward
	}

	return nil
}

// ShouldIgnoreGasUsed returns true if the gasUsed in result should be ignored
// workaround for issue: https://github.com/cosmos/cosmos-sdk/issues/10832
func ShouldIgnoreGasUsed(res *abci.ExecTxResult) bool {
	return res.GetCode() == 11 && strings.Contains(res.GetLog(), "no block gas left to run tx: out of gas")
}

// GetLogsFromBlockResults returns the list of event logs from the comet block result response
func GetLogsFromBlockResults(blockRes *cmrpctypes.ResultBlockResults) ([][]*ethtypes.Log, error) {
	blockLogs := [][]*ethtypes.Log{}
	for _, txResult := range blockRes.TxResults {
		logs, err := evmtypes.DecodeTxLogsFromEvents(txResult.Data, txResult.Events, uint64(blockRes.Height))
		if err != nil {
			return nil, err
		}
		blockLogs = append(blockLogs, logs)
	}
	return blockLogs, nil
}

// GetHexProofs returns list of hex data of proof op
func GetHexProofs(proof *crypto.ProofOps) []string {
	if proof == nil {
		return []string{""}
	}
	proofs := []string{}
	// check for proof
	for _, p := range proof.Ops {
		proof := ""
		if len(p.Data) > 0 {
			proof = hexutil.Encode(p.Data)
		}
		proofs = append(proofs, proof)
	}
	return proofs
}
