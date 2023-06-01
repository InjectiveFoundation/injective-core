package ante

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	log "github.com/xlab/suplog"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
)

const (
	// TODO: Use this cost per byte through parameter or overriding NewConsumeGasForTxSizeDecorator
	// which currently defaults at 10, if intended
	// memoCostPerByte     sdk.Gas = 3
	secp256k1VerifyCost uint64 = 21000
)

// AccountKeeper defines an expected keeper interface for the auth module's AccountKeeper
type AccountKeeper interface {
	NewAccount(sdk.Context, authtypes.AccountI) authtypes.AccountI
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI

	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI
	GetAllAccounts(ctx sdk.Context) []authtypes.AccountI
	SetAccount(ctx sdk.Context, acc authtypes.AccountI)

	IterateAccounts(ctx sdk.Context, process func(authtypes.AccountI) bool)

	ValidatePermissions(macc authtypes.ModuleAccountI) error

	GetModuleAddress(moduleName string) sdk.AccAddress
	GetModuleAddressAndPermissions(moduleName string) (addr sdk.AccAddress, permissions []string)
	GetModuleAccountAndPermissions(ctx sdk.Context, moduleName string) (authtypes.ModuleAccountI, []string)
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	SetModuleAccount(ctx sdk.Context, macc authtypes.ModuleAccountI)

	authante.AccountKeeper
}

// BankKeeper defines an expected keeper interface for the bank module's Keeper
type BankKeeper interface {
	authtypes.BankKeeper
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// FeegrantKeeper defines an expected keeper interface for the feegrant module's Keeper
type FeegrantKeeper interface {
	UseGrantedFees(ctx sdk.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

// NewAnteHandler returns an ante handler responsible for attempting to route an
// Ethereum or SDK transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto it's respective handler.
func NewAnteHandler(
	ak AccountKeeper,
	bankKeeper BankKeeper,
	feegrantKeeper FeegrantKeeper,
	signModeHandler authsigning.SignModeHandler,
	txCounterStoreKey storetypes.StoreKey,
	wasmConfig wasmTypes.WasmConfig,
	ibcKeeper *ibckeeper.Keeper,
) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/injective.evm.v1beta1.ExtensionOptionsEthereumTx":
					return ctx, errors.Wrap(sdkerrors.ErrUnknownRequest, "ExtensionOptionsEthereumTx is not supported by this instance")
				case "/injective.types.v1beta1.ExtensionOptionsWeb3Tx":
					// handle as normal Cosmos SDK tx, except signature is checked for EIP712 representation

					switch tx.(type) {
					case sdk.Tx:
						anteHandler = sdk.ChainAnteDecorators(
							authante.NewSetUpContextDecorator(),                                      // outermost AnteDecorator. SetUpContext must be called first
							wasmkeeper.NewLimitSimulationGasDecorator(wasmConfig.SimulationGasLimit), // after setup context to enforce limits early
							wasmkeeper.NewCountTXDecorator(txCounterStoreKey),
							authante.NewValidateBasicDecorator(),
							authante.NewTxTimeoutHeightDecorator(),
							authante.NewValidateMemoDecorator(ak),
							authante.NewConsumeGasForTxSizeDecorator(ak),
							authante.NewSetPubKeyDecorator(ak), // SetPubKeyDecorator must be called before all signature verification decorators
							authante.NewValidateSigCountDecorator(ak),
							NewDeductFeeDecorator(ak, bankKeeper), // overidden for fee delegation
							authante.NewSigGasConsumeDecorator(ak, DefaultSigVerificationGasConsumer),
							NewEip712SigVerificationDecorator(ak, signModeHandler), // overidden for EIP712 Tx signatures
							authante.NewIncrementSequenceDecorator(ak),             // innermost AnteDecorator
						)
					default:
						return ctx, errors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
					}

				default:
					log.WithField("type_url", typeURL).Errorln("rejecting tx with unsupported extension option")
					return ctx, sdkerrors.ErrUnknownExtensionOptions
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx

		switch tx.(type) {
		case sdk.Tx:
			anteHandler = sdk.ChainAnteDecorators(
				authante.NewSetUpContextDecorator(),                                      // outermost AnteDecorator. SetUpContext must be called first
				wasmkeeper.NewLimitSimulationGasDecorator(wasmConfig.SimulationGasLimit), // after setup context to enforce limits early
				wasmkeeper.NewCountTXDecorator(txCounterStoreKey),
				authante.NewExtensionOptionsDecorator(nil),
				authante.NewValidateBasicDecorator(),
				authante.NewTxTimeoutHeightDecorator(),
				authante.NewValidateMemoDecorator(ak),
				authante.NewConsumeGasForTxSizeDecorator(ak),
				authante.NewDeductFeeDecorator(ak, bankKeeper, feegrantKeeper, nil),
				authante.NewSetPubKeyDecorator(ak), // SetPubKeyDecorator must be called before all signature verification decorators
				authante.NewValidateSigCountDecorator(ak),
				authante.NewSigGasConsumeDecorator(ak, DefaultSigVerificationGasConsumer),
				authante.NewSigVerificationDecorator(ak, signModeHandler),
				authante.NewIncrementSequenceDecorator(ak),
				ibcante.NewRedundantRelayDecorator(ibcKeeper),
			)
		default:
			return ctx, errors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}
}

var _ = DefaultSigVerificationGasConsumer

// DefaultSigVerificationGasConsumer is the default implementation of SignatureVerificationGasConsumer. It consumes gas
// for signature verification based upon the public key type. The cost is fetched from the given params and is matched
// by the concrete type.
func DefaultSigVerificationGasConsumer(
	meter sdk.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	pubkey := sig.PubKey
	switch pubkey := pubkey.(type) {
	case *ed25519.PubKey:
		meter.ConsumeGas(params.SigVerifyCostED25519, "ante verify: ed25519")
		return nil

	case *secp256k1.PubKey:
		meter.ConsumeGas(params.SigVerifyCostSecp256k1, "ante verify: secp256k1")
		return nil

	// support for ethereum ECDSA secp256k1 keys
	case *ethsecp256k1.PubKey:
		meter.ConsumeGas(secp256k1VerifyCost, "ante verify: eth_secp256k1")
		return nil

	case multisig.PubKey:
		multisignature, ok := sig.Data.(*signing.MultiSignatureData)
		if !ok {
			return fmt.Errorf("expected %T, got, %T", &signing.MultiSignatureData{}, sig.Data)
		}
		err := ConsumeMultisignatureVerificationGas(meter, multisignature, pubkey, params, sig.Sequence)
		if err != nil {
			return err
		}
		return nil

	default:
		return errors.Wrapf(sdkerrors.ErrInvalidPubKey, "unrecognized public key type: %T", pubkey)
	}
}

// ConsumeMultisignatureVerificationGas consumes gas from a GasMeter for verifying a multisig pubkey signature
func ConsumeMultisignatureVerificationGas(
	meter sdk.GasMeter, sig *signing.MultiSignatureData, pubkey multisig.PubKey,
	params authtypes.Params, accSeq uint64,
) error {

	size := sig.BitArray.Count()
	sigIndex := 0

	for i := 0; i < size; i++ {
		if !sig.BitArray.GetIndex(i) {
			continue
		}
		sigV2 := signing.SignatureV2{
			PubKey:   pubkey.GetPubKeys()[i],
			Data:     sig.Signatures[sigIndex],
			Sequence: accSeq,
		}
		err := DefaultSigVerificationGasConsumer(meter, sigV2, params)
		if err != nil {
			return err
		}
		sigIndex++
	}

	return nil
}
