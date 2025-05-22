package tx

import (
	"math/big"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func CreateContractMsgTx(
	nonce uint64,
	signer ethtypes.Signer,
	gasPrice *big.Int,
	gas uint64,
	from common.Address,
	keyringSigner keyring.Signer,
) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      gas,
		To:       nil,
		Data:     []byte("contract_data"),
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = from.Bytes()

	err := ethMsg.Sign(signer, keyringSigner)
	if err != nil {
		return nil, err
	}

	return ethMsg, nil
}

func CreateRevertingContractMsgTx(
	nonce uint64,
	signer ethtypes.Signer,
	gasPrice *big.Int,
	from common.Address,
	keyringSigner keyring.Signer,
) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      params.TxGasContractCreation + 136, // accurate accounting, since test cannot refund
		To:       nil,
		Data:     common.Hex2Bytes("deadbeef"),
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = from.Bytes()

	err := ethMsg.Sign(signer, keyringSigner)
	if err != nil {
		return nil, err
	}

	return ethMsg, nil
}

func CreateNoCodeCallMsgTx(
	nonce uint64,
	signer ethtypes.Signer,
	gasPrice *big.Int,
	from common.Address,
	keyringSigner keyring.Signer,
) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      params.TxGas + params.TxDataNonZeroGasEIP2028*4, // accurate accounting, since test cannot refund
		To:       &common.Address{},                               // no code exists - considered EOA
		Data:     common.Hex2Bytes("deadbeef"),                    // 4byte selector
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = from.Bytes()

	err := ethMsg.Sign(signer, keyringSigner)
	if err != nil {
		return nil, err
	}

	return ethMsg, nil
}
