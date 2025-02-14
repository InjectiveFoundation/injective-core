package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

type Tally struct {
	AbstainCount    math.Int `json:"abstain_count"`
	NoCount         math.Int `json:"no_count"`
	NoWithVetoCount math.Int `json:"no_with_veto_count"`
	YesCount        math.Int `json:"yes_count"`
}

const (
	ProposalVoteYes        = "yes"
	ProposalVoteNo         = "no"
	ProposalVoteNoWithVeto = "no_with_veto"
	ProposalVoteAbstain    = "abstain"
)

// QueryProposalTally gets tally results for a proposal
func QueryProposalTally(t *testing.T, ctx context.Context, chainNode *cosmos.ChainNode, proposalID string) Tally {
	stdout, _, err := chainNode.ExecQuery(ctx, "gov", "tally", proposalID)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	var tally Tally
	err = json.Unmarshal([]byte(stdout), &tally)
	require.NoError(t, err)

	return tally
}

// LegacyTextProposal submits a text governance proposal to the chain.
func LegacyTextProposal(ctx context.Context, keyName string, chainNode *cosmos.ChainNode, prop cosmos.TextProposal) (string, error) {
	command := []string{
		"gov", "submit-legacy-proposal",
		"--type", "text",
		"--title", prop.Title,
		"--description", prop.Description,
		"--deposit", prop.Deposit,
	}
	if prop.Expedited {
		command = append(command, "--is-expedited=true")
	}

	return chainNode.ExecTx(ctx, keyName, command...)
}

// QueryProposalTx reads results of a proposal Tx, useful to get the ProposalID
func QueryProposalTx(ctx context.Context, chainNode *cosmos.ChainNode, txHash string) (tx cosmos.TxProposal, _ error) {
	txResp, err := getTxResponse(ctx, chainNode, txHash)
	if err != nil {
		return tx, fmt.Errorf("failed to get transaction %s: %w", txHash, err)
	}

	if txResp.Code != 0 {
		return tx, fmt.Errorf("proposal transaction error: code %d %s", txResp.Code, txResp.RawLog)
	}

	tx.Height = txResp.Height
	tx.TxHash = txHash
	// In cosmos, user is charged for entire gas requested, not the actual gas used.
	tx.GasSpent = txResp.GasWanted
	events := txResp.Events

	tx.DepositAmount, _ = cometAttributeValue(events, "proposal_deposit", "amount")

	evtSubmitProp := "submit_proposal"
	tx.ProposalID, _ = cometAttributeValue(events, evtSubmitProp, "proposal_id")
	tx.ProposalType, _ = cometAttributeValue(events, evtSubmitProp, "proposal_type")

	return tx, nil
}
