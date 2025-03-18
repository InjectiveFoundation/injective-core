package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// QueryAllValidators lists all validators
func QueryAllValidators(t *testing.T, ctx context.Context, chainNode *cosmos.ChainNode) []Validator {
	stdout, _, err := chainNode.ExecQuery(ctx, "staking", "validators")
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	var resp queryValidatorsResponse
	err = json.Unmarshal([]byte(stdout), &resp)
	require.NoError(t, err)

	return resp.Validators
}

// QueryValidator gets info about particular validator
func QueryValidator(
	t *testing.T,
	ctx context.Context,
	chainNode *cosmos.ChainNode,
	valoperAddr string,
) Validator {
	stdout, _, err := chainNode.ExecQuery(ctx, "staking", "validator", valoperAddr)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	var validator Validator
	err = json.Unmarshal([]byte(stdout), &validator)
	require.NoError(t, err)

	return validator
}

// QueryDelegation gets info about particular delegation
func QueryDelegation(
	t *testing.T,
	ctx context.Context,
	chainNode *cosmos.ChainNode,
	delegatorAddr string,
	valoperAddr string,
) Delegation {
	stdout, _, err := chainNode.ExecQuery(ctx, "staking", "delegation", delegatorAddr, valoperAddr)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	var resp queryDelegationResponse
	err = json.Unmarshal([]byte(stdout), &resp)
	require.NoError(t, err)

	return resp.Delegation
}

type queryDelegationResponse struct {
	Delegation Delegation `json:"delegation"`
}

type Delegation struct {
	DelegatorAddress string         `json:"delegator_address"`
	ValidatorAddress string         `json:"validator_address"`
	Shares           math.LegacyDec `json:"shares"`
}

type queryValidatorsResponse struct {
	Validators []Validator `json:"validators"`

	Pagination struct {
		Total string `json:"total"`
	} `json:"pagination"`
}

type Validator struct {
	OperatorAddress string `json:"operator_address"`

	ConsensusPubkey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"consensus_pubkey"`

	Status          string `json:"status"`
	Tokens          string `json:"tokens"`
	DelegatorShares string `json:"delegator_shares"`

	Description struct {
		Moniker string `json:"moniker"`
	} `json:"description"`

	UnbondingTime time.Time `json:"unbonding_time"`

	Commission struct {
		CommissionRates struct {
			Rate          string `json:"rate"`
			MaxRate       string `json:"max_rate"`
			MaxChangeRate string `json:"max_change_rate"`
		} `json:"commission_rates"`
		UpdateTime time.Time `json:"update_time"`
	} `json:"commission"`

	MinSelfDelegation string `json:"min_self_delegation"`
}
