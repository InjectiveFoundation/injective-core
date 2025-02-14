package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

// AttestationHandler processes `observed` Attestations
type AttestationHandler struct {
	keeper     Keeper
	bankKeeper types.BankKeeper
	svcTags    metrics.Tags
}

func NewAttestationHandler(bankKeeper types.BankKeeper, keeper Keeper) AttestationHandler {
	return AttestationHandler{
		keeper:     keeper,
		bankKeeper: bankKeeper,
		svcTags: metrics.Tags{
			"svc": "peggy_att",
		},
	}
}

// Handle is the entry point for Attestation processing.
func (h AttestationHandler) Handle(ctx sdk.Context, claim types.EthereumClaim) error {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	switch claim := claim.(type) {
	case *types.MsgDepositClaim:
		return h.handleDepositClaim(ctx, claim)
	case *types.MsgWithdrawClaim:
		h.handleWithdrawClaim(ctx, claim)
	case *types.MsgERC20DeployedClaim:
		return h.handleERC20DeployedClaim(ctx, claim)
	case *types.MsgValsetUpdatedClaim:
		h.handleValsetUpdatedClaim(ctx, claim)
	default:
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(types.ErrInvalid, fmt.Sprintf("Invalid event type for attestations %s", claim.GetType()))
	}

	return nil
}

func (h AttestationHandler) handleDepositClaim(ctx sdk.Context, claim *types.MsgDepositClaim) error {
	sender, err := types.NewEthAddress(claim.EthereumSender)
	if err != nil {
		// likewise nil sender would have to be caused by a bogus event
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(err, "failed to parse ethereum sender in claim")
	}

	// Check if coin is Cosmos-originated asset and get denom
	isCosmosOriginated, denom := h.keeper.ERC20ToDenomLookup(ctx, common.HexToAddress(claim.TokenContract))
	depositCoin := sdk.NewCoin(denom, claim.Amount)

	if !isCosmosOriginated {
		// Check if supply overflows with claim amount
		currentSupply := h.bankKeeper.GetSupply(ctx, denom)
		newSupply := new(big.Int).Add(currentSupply.Amount.BigInt(), claim.Amount.BigInt())
		if newSupply.BitLen() > 256 {
			metrics.ReportFuncError(h.svcTags)
			return errors.Wrap(types.ErrSupplyOverflow, "invalid coin supply")
		}

		if err := h.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(depositCoin)); err != nil {
			metrics.ReportFuncError(h.svcTags)
			return errors.Wrapf(err, "failed to mint deposit coin: %s", depositCoin.String())
		}
	}

	receiver, err := sdk.AccAddressFromBech32(claim.CosmosReceiver)
	if err != nil {
		// #1: receiver address is malformed, deposit into community pool
		if err := h.keeper.SendToCommunityPool(ctx, sdk.NewCoins(depositCoin)); err != nil {
			return errors.Wrap(err, "failed to send deposit to community pool")
		}

		receiver = h.keeper.accountKeeper.GetModuleAccount(ctx, distrtypes.ModuleName).GetAddress()
		_ = ctx.EventManager().EmitTypedEvent(types.NewEventDepositReceived(*sender, receiver, depositCoin))
		return nil
	}

	if h.keeper.IsOnBlacklist(ctx, *sender) {
		// #2: sender is blacklister, we deposit to segregated wallet
		receiver = sdk.MustAccAddressFromBech32(h.keeper.GetParams(ctx).SegregatedWalletAddress)
		if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, sdk.NewCoins(depositCoin)); err != nil {
			return errors.Wrap(err, "failed to send sanctioned deposit to segregated wallet")
		}

		_ = ctx.EventManager().EmitTypedEvent(types.NewEventDepositReceived(*sender, receiver, depositCoin))
		return nil
	}

	// #3: address appears valid, attempt to send minted/locked coins to receiver
	if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, sdk.NewCoins(depositCoin)); err != nil {
		// last attempt (default behavior)
		if err := h.keeper.SendToCommunityPool(ctx, sdk.NewCoins(depositCoin)); err != nil {
			return errors.Wrap(err, "failed to send deposit to community pool")
		}

		receiver = h.keeper.accountKeeper.GetModuleAccount(ctx, distrtypes.ModuleName).GetAddress()
	}

	_ = ctx.EventManager().EmitTypedEvent(types.NewEventDepositReceived(*sender, receiver, depositCoin))
	return nil
}

func (h AttestationHandler) handleWithdrawClaim(ctx sdk.Context, claim *types.MsgWithdrawClaim) {
	h.keeper.OutgoingTxBatchExecuted(ctx, common.HexToAddress(claim.TokenContract), claim.BatchNonce)
}

func (h AttestationHandler) handleERC20DeployedClaim(ctx sdk.Context, claim *types.MsgERC20DeployedClaim) error {
	// Check if it already exists
	existingERC20, exists := h.keeper.GetCosmosOriginatedERC20(ctx, claim.CosmosDenom)
	if exists {
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(types.ErrInvalid, fmt.Sprintf("ERC20 %s already exists for denom %s", existingERC20, claim.CosmosDenom))
	}

	// Check if denom exists
	metadata, found := h.keeper.bankKeeper.GetDenomMetaData(ctx, claim.CosmosDenom)
	if metadata.Base == "" || !found {
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(types.ErrUnknown, fmt.Sprintf("denom not found %s", claim.CosmosDenom))
	}

	// Check if attributes of ERC20 match Cosmos denom
	if claim.Name != metadata.Display {
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(
			types.ErrInvalid,
			fmt.Sprintf("ERC20 name %s does not match denom display %s", claim.Name, metadata.Description))
	}

	if claim.Symbol != metadata.Display {
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(
			types.ErrInvalid,
			fmt.Sprintf("ERC20 symbol %s does not match denom display %s", claim.Symbol, metadata.Display))
	}

	// ERC20 tokens use a very simple mechanism to tell you where to display the decimal point.
	// The "decimals" field simply tells you how many decimal places there will be.
	// Cosmos denoms have a system that is much more full featured, with enterprise-ready token denominations.
	// There is a DenomUnits array that tells you what the name of each denomination of the
	// token is.
	// To correlate this with an ERC20 "decimals" field, we have to search through the DenomUnits array
	// to find the DenomUnit which matches up to the main token "display" value. Then we take the
	// "exponent" from this DenomUnit.
	// If the correct DenomUnit is not found, it will default to 0. This will result in there being no decimal places
	// in the token's ERC20 on Ethereum. So, for example, if this happened with Atom, 1 Atom would appear on Ethereum
	// as 1 million Atoms, having 6 extra places before the decimal point.
	// This will only happen with a Denom Metadata which is for all intents and purposes invalid, but I am not sure
	// this is checked for at any other point.
	decimals := uint32(0)
	for _, denomUnit := range metadata.DenomUnits {
		if denomUnit.Denom == metadata.Display {
			decimals = denomUnit.Exponent
			break
		}
	}

	if uint64(decimals) != claim.Decimals {
		metrics.ReportFuncError(h.svcTags)
		return errors.Wrap(
			types.ErrInvalid,
			fmt.Sprintf("ERC20 decimals %d does not match denom decimals %d", claim.Decimals, decimals))
	}

	// Add to denom-erc20 mapping
	h.keeper.SetCosmosOriginatedDenomToERC20(ctx, claim.CosmosDenom, common.HexToAddress(claim.TokenContract))

	return nil
}

func (h AttestationHandler) handleValsetUpdatedClaim(ctx sdk.Context, claim *types.MsgValsetUpdatedClaim) {
	// TODO here we should check the contents of the validator set against
	// the store, if they differ we should take some action to indicate to the
	// user that bridge highjacking has occurred
	h.keeper.SetLastObservedValset(ctx, types.Valset{
		Nonce:        claim.ValsetNonce,
		Members:      claim.Members,
		RewardAmount: claim.RewardAmount,
		RewardToken:  claim.RewardToken,
	})
}
