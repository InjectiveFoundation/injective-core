package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewDefaultPolicyStatuses returns a list of permissive default policy statuses for all actions
func NewDefaultPolicyStatuses() []*PolicyStatus {
	policyStatuses := make([]*PolicyStatus, 0, len(Actions))
	for _, action := range Actions {
		status := NewPolicyStatus(action, false, false)
		policyStatuses = append(policyStatuses, status)
	}

	return policyStatuses
}

// NewDefaultPolicyManagerCapabilities returns a list of permissive default policy manager capabilities for all actions
// for the given creator
func NewDefaultPolicyManagerCapabilities(creator sdk.AccAddress) []*PolicyManagerCapability {
	capabilities := make([]*PolicyManagerCapability, 0, len(Actions))

	for _, action := range Actions {
		capability := NewPolicyManagerCapability(creator, action, true, true)
		capabilities = append(capabilities, capability)
	}
	return capabilities
}
