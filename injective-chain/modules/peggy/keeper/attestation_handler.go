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
		// todo: upgrade Peggy.sol on testnet
		// The deployERC20 functionality was removed from mainnet contract.
		// Logic below no longer applies so we return early with no error
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
	tokenContract := common.HexToAddress(claim.TokenContract)
	isCosmosOriginated, denom := h.keeper.ERC20ToDenomLookup(ctx, tokenContract)
	depositCoin := sdk.NewCoin(denom, claim.Amount)

	rateLimit := h.keeper.GetRateLimit(ctx, tokenContract)
	withRateLimit := rateLimit != nil
	currentMintAmount := h.keeper.GetMintAmountERC20(ctx, tokenContract)

	if !isCosmosOriginated {
		// Check if supply overflows with claim amount
		currentSupply := h.bankKeeper.GetSupply(ctx, denom)
		newSupply := new(big.Int).Add(currentSupply.Amount.BigInt(), claim.Amount.BigInt())
		if newSupply.BitLen() > 256 {
			metrics.ReportFuncError(h.svcTags)
			return errors.Wrap(types.ErrSupplyOverflow, "invalid coin supply")
		}

		// check absolute limit
		if withRateLimit {
			absoluteLimit := rateLimit.AbsoluteMintLimit.BigInt()
			if remaining := absoluteLimit.Sub(absoluteLimit, currentMintAmount); remaining.Cmp(claim.Amount.BigInt()) < 0 {
				return ErrAbsoluteMintLimitOverflow
			}
		}

		if err := h.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(depositCoin)); err != nil {
			metrics.ReportFuncError(h.svcTags)
			return errors.Wrapf(err, "failed to mint deposit coin: %s", depositCoin.String())
		}

		// track new mint
		if withRateLimit {
			newAmount := currentMintAmount.Add(currentMintAmount, claim.Amount.BigInt())
			h.keeper.SetMintAmountERC20(ctx, tokenContract, newAmount)
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

	h.keeper.TrackTokenInflow(ctx, tokenContract, depositCoin.Amount)

	_ = ctx.EventManager().EmitTypedEvent(types.NewEventDepositReceived(*sender, receiver, depositCoin))
	return nil
}

func (h AttestationHandler) handleWithdrawClaim(ctx sdk.Context, claim *types.MsgWithdrawClaim) {
	h.keeper.OutgoingTxBatchExecuted(ctx, common.HexToAddress(claim.TokenContract), claim.BatchNonce)
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
