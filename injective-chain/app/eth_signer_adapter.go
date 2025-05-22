package app

import (
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
)

var _ signerextraction.Adapter = EthSignerExtractionAdapter{}

// EthSignerExtractionAdapter is the default implementation of SignerExtractionAdapter. It extracts the signers
// from a cosmos-sdk tx via GetSignaturesV2.
type EthSignerExtractionAdapter struct {
	fallback signerextraction.Adapter
}

// NewEthSignerExtractionAdapter constructs a new EthSignerExtractionAdapter instance
func NewEthSignerExtractionAdapter(fallback signerextraction.Adapter) EthSignerExtractionAdapter {
	return EthSignerExtractionAdapter{fallback}
}

// GetSigners implements the Adapter interface
// NOTE: only the first item is used by the blocksdk mempool
func (s EthSignerExtractionAdapter) GetSigners(tx sdk.Tx) ([]signerextraction.SignerData, error) {
	if txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx); ok {
		opts := txWithExtensions.GetExtensionOptions()
		if len(opts) > 0 && opts[0].GetTypeUrl() == "/injective.evm.v1.ExtensionOptionsEthereumTx" {
			for _, msg := range tx.GetMsgs() {
				if ethMsg, ok := msg.(*evmtypes.MsgEthereumTx); ok {
					return []signerextraction.SignerData{
						signerextraction.NewSignerData(
							ethMsg.GetFrom(),
							ethMsg.AsTransaction().Nonce(),
						),
					}, nil
				}
			}
		}
	}

	return s.fallback.GetSigners(tx)
}
