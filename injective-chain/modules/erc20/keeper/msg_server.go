package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

var (
	symbolRetType, _ = abi.NewType("string", "string", nil)
	symbolMethod     = abi.NewMethod("symbol", "symbol", abi.Function, "", false, false, abi.Arguments{}, abi.Arguments{abi.Argument{
		Type: symbolRetType,
	}})
	ERC20SymbolCallInput = hexutil.Bytes(symbolMethod.ID)
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) CreateTokenPair(c context.Context, msg *types.MsgCreateTokenPair) (*types.MsgCreateTokenPairResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	bankDenom := msg.TokenPair.BankDenom
	erc20Address := common.HexToAddress(msg.TokenPair.Erc20Address)

	// validate that the pair doesn't already exist
	if pair, _ := k.GetTokenPairForDenom(ctx, bankDenom); pair != nil {
		return nil, errors.Wrapf(types.ErrTokenPairExists, "token pair for denom %s already exists", bankDenom)
	}

	if pair, _ := k.GetTokenPairForERC20(ctx, erc20Address); pair != nil {
		return nil, errors.Wrapf(types.ErrTokenPairExists, "token pair for ERC20 token %s already exists", erc20Address)
	}

	// validate that bank denom exists
	if !k.bankKeeper.HasSupply(c, bankDenom) {
		return nil, types.ErrUnknownBankDenom
	}

	// if ERC20 address is defined, check that it exists and is, in fact, an ERC20 contract
	if msg.TokenPair.Erc20Address != "" {
		if err := k.validateErc20Address(ctx, erc20Address); err != nil {
			return nil, err
		}
	}

	pair := msg.TokenPair // copy request token pair
	if err := k.createTokenPair(ctx, sdk.MustAccAddressFromBech32(msg.Sender), &pair); err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCreateTokenPair{
		BankDenom:    pair.BankDenom,
		Erc20Address: pair.Erc20Address,
	})

	return &types.MsgCreateTokenPairResponse{
		TokenPair: pair,
	}, nil
}

func (k Keeper) validateErc20Address(ctx sdk.Context, erc20Address common.Address) error {
	// does account exist?
	if acc := k.evmKeeper.GetAccount(ctx, erc20Address); acc == nil || bytes.Equal(acc.CodeHash, evmtypes.EmptyCodeHash) {
		return errors.Wrap(types.ErrInvalidTokenPair, "ERC20 contract address is not correct or doesn't exist")
	}
	// check that the SC does not have associated "erc20:..." token circualating already
	erc20Denom := fmt.Sprintf(types.DenomPrefix + erc20Address.String())
	if k.bankKeeper.HasSupply(ctx, erc20Denom) {
		return errors.Wrapf(types.ErrExistingERC20DenomSupply, "smart contract has circulating supply of denom %s", erc20Denom)
	}

	// now check that contract is ERC20 (has symbol() function)
	args, _ := json.Marshal(evmtypes.TransactionArgs{To: &erc20Address, Input: &ERC20SymbolCallInput})
	resp, err := k.evmKeeper.EthCall(ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: uint64(300_000),
	})
	if err != nil || resp.VmError != "" {
		var errText string
		if err != nil {
			errText = err.Error()
		} else {
			errText = resp.VmError
		}
		return errors.Wrapf(types.ErrInvalidERC20Address, "ERC20 token address is not a valid ERC20 smart contract: %s", errText)
	}
	return nil
}

func (k msgServer) DeleteTokenPair(c context.Context, msg *types.MsgDeleteTokenPair) (*types.MsgDeleteTokenPairResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	pair, _ := k.GetTokenPairForDenom(ctx, msg.BankDenom)
	if pair == nil {
		return nil, errors.Wrapf(types.ErrUnknownBankDenom, "token pair for denom %s does not exist", msg.BankDenom)
	}
	if msg.Sender != k.authority {
		return nil, errors.Wrapf(types.ErrUnauthorized, "invalid sender: expected %s, got %s", k.authority, msg.Sender)
	}

	k.deleteTokenPair(ctx, *pair)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventDeleteTokenPair{
		BankDenom: pair.BankDenom,
	})

	return &types.MsgDeleteTokenPairResponse{}, nil
}
