package helpers

import (
	"context"
	"encoding/json"
	"math/big"
	"regexp"
	"strings"

	"github.com/InjectiveLabs/sdk-go/chain/crypto/ethsecp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

func SignAndBroadcastEthTx(
	ctx context.Context,
	chainNode *cosmos.ChainNode,
	ethChainID *big.Int,
	fromName string,
	fromPrivKey cryptotypes.PrivKey,
	legacyTx *ethtypes.LegacyTx,
	checkTxError bool,
) (
	cosmosTxHash string,
	ethTxHash ethcmn.Hash,
	err error,
) {
	tx := ethtypes.NewTx(legacyTx)

	ethSigner := ethtypes.LatestSignerForChainID(ethChainID)
	txHashToSign := ethSigner.Hash(tx)

	ethPrivKey, ok := fromPrivKey.(*ethsecp256k1.PrivKey)
	if !ok {
		err = errors.Errorf("failed to convert privKey to ethsecp256k1.PrivKey: got %T", fromPrivKey)
		return "", ethcmn.Hash{}, err
	}

	sig, err := ethcrypto.Sign(txHashToSign.Bytes(), ethPrivKey.ToECDSA())
	if err != nil {
		err = errors.Wrap(err, "failed to sign Ethereum Tx hash")
		return "", ethcmn.Hash{}, err
	}

	tx, err = tx.WithSignature(ethSigner, sig)
	if err != nil {
		err = errors.Wrap(err, "failed to update Ethereum Tx with signature")
		return "", ethcmn.Hash{}, err
	}

	ethTxHash = tx.Hash()
	cosmosTxHash, err = broadcastSignedEthTx(ctx, chainNode, fromName, tx, checkTxError)
	return cosmosTxHash, ethTxHash, err
}

func broadcastSignedEthTx(
	ctx context.Context,
	chainNode *cosmos.ChainNode,
	fromName string,
	signedTx *ethtypes.Transaction,
	checkTxError bool,
) (
	cosmosTxHash string,
	err error,
) {
	txData, err := signedTx.MarshalBinary()
	if err != nil {
		err = errors.Wrap(err, "failed to binary marshal signed Ethereum Tx")
		return "", err
	}

	// if checkTxError, the built-in ExecTx is fine

	if checkTxError {
		if cosmosTxHash, err = chainNode.ExecTx(
			ctx, fromName, "evm", "raw",
			hexutil.Encode(txData),
		); err != nil {
			err = errors.Wrap(err, "failed to broadcast signed Ethereum Tx")
			return "", err
		}

		return cosmosTxHash, nil
	}

	// or, we need to broadcast and not check the execution error here

	stdout, stderr, err := chainNode.Exec(ctx,
		chainNode.TxCommand(
			fromName, "evm", "raw", hexutil.Encode(txData),
		),
		chainNode.Chain.Config().Env,
	)
	if err != nil {
		err = errors.Wrap(err, "failed to broadcast signed Ethereum Tx")
		return "", err
	} else if len(stderr) != 0 {
		err = errors.Errorf("failed to broadcast signed Ethereum Tx: %s", string(stderr))
		return "", err
	}

	var out broadcastStdoutJSON
	if err = json.Unmarshal(stdout, &out); err != nil {
		err = errors.Wrap(err, "failed to parse stdout of broadcast signed Ethereum Tx")
		return "", err
	}

	return out.TxHash, nil
}

// {"height":"0","txhash":"E7C4151C31ABBB81663D46BC32499C82C1AE63F505CBAAEC122DD200ACC2236F","codespace":"","code":0,"data":"","raw_log":"","logs":[],"info":"","gas_wanted":"0","gas_used":"0","tx":null,"timestamp":"","events":[]}

type broadcastStdoutJSON struct {
	TxHash string `json:"txhash"`
}

var (
	injectiveEthChainIDRx = regexp.MustCompile(`^([a-z]{1,})-{1}([1-9][0-9]*)$`)
)

// ParseEthChainID parses a string chain identifier's epoch to an Ethereum-compatible
// chain-id in *big.Int format. The function returns an error if the chain-id has an invalid format
//
// NOTE: This function is copied from Ethermint's internal utils. To avoid dependency.
func ParseEthChainID(chainID string) (*big.Int, error) {
	chainID = strings.TrimSpace(chainID)
	if len(chainID) > 48 {
		return nil, errors.Errorf("chain-id '%s' cannot exceed 48 chars", chainID)
	}

	matches := injectiveEthChainIDRx.FindStringSubmatch(chainID)
	if matches == nil || len(matches) != 3 || matches[1] == "" {
		return nil, errors.Errorf("%s: %v", chainID, matches)
	}

	// verify that the chain-id entered is a base 10 integer
	chainIDInt, ok := new(big.Int).SetString(matches[2], 10)
	if !ok {
		return nil, errors.Errorf("epoch %s must be base-10 integer format", matches[2])
	}

	return chainIDInt, nil
}
