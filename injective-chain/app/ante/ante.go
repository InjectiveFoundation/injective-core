package ante

import (
	"fmt"
	"os"

	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	evmante "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/ante"
	evmkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/keeper"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	txfeeskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper"
)

const (
	// TODO: Use this cost per byte through parameter or overriding NewConsumeGasForTxSizeDecorator
	// which currently defaults at 10, if intended
	// memoCostPerByte     sdk.Gas = 3
	secp256k1VerifyCost uint64 = 21000
)

// BankKeeper defines an expected keeper interface for the bank module's Keeper
type BankKeeper interface {
	authtypes.BankKeeper
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// FeegrantKeeper defines an expected keeper interface for the feegrant module's Keeper
type FeegrantKeeper interface {
	UseGrantedFees(ctx sdk.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	authante.HandlerOptions

	IBCKeeper             *ibckeeper.Keeper
	WasmConfig            *wasmtypes.WasmConfig
	WasmKeeper            *wasmkeeper.Keeper
	TXCounterStoreService corestoretypes.KVStoreService
	TxFeesKeeper          *txfeeskeeper.Keeper
	EVMKeeper             *evmkeeper.Keeper
	MaxEthTxGasWanted     uint64
	DisabledAuthzMsgs     []string
}

func newEVMAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	return evmante.NewEthAnteHandler(evmante.EthAnteHandlerOptions{
		// authante.HandlerOptions has more relaxed requirements for the keepers,
		// re-cast into the expected interfaces.
		AccountKeeper: options.AccountKeeper.(evmtypes.AccountKeeper),
		BankKeeper:    options.BankKeeper.(evmtypes.BankKeeper),

		EvmKeeper:       options.EVMKeeper,
		SignModeHandler: options.SignModeHandler,
		SigGasConsumer:  options.SigGasConsumer,
		MaxTxGasWanted:  options.MaxEthTxGasWanted,
		TxFeesDecorator: txfeeskeeper.NewMempoolFeeDecorator(options.TxFeesKeeper, true),
		// TODO: use TxFeesKeeper
		DisabledAuthzMsgs: []string{
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		},
	})
}

type noopAnteDecorator struct{}

func (n noopAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler) (sdk.Context, error) {
	return ctx, nil
}

var _ sdk.AnteDecorator = noopAnteDecorator{}

// NewAnteHandler returns an ante handler responsible for attempting to route an
// Ethereum or SDK transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto it's respective handler.
func NewAnteHandler(
	options HandlerOptions,
	// ak AccountKeeper,
	// bankKeeper BankKeeper,
	// feegrantKeeper FeegrantKeeper,
	// signModeHandler authsigning.SignModeHandler,
	// txCounterStoreKey storetypes.StoreKey,
	// wasmConfig wasmtypes.WasmConfig,
	// ibcKeeper *ibckeeper.Keeper,
) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		ak := options.AccountKeeper

		noSignatureVerification := os.Getenv("DEVNET_NO_SIGNATURE_VERIFICATION") == "true" ||
			os.Getenv("DEVNET_NO_SIGNATURE_VERIFICATION") == "1"

		var eip712SigVerificationDecorator sdk.AnteDecorator = NewEip712SigVerificationDecorator(ak)
		if noSignatureVerification {
			eip712SigVerificationDecorator = noopAnteDecorator{}
		}

		var sigVerificationDecorator sdk.AnteDecorator = authante.NewSigVerificationDecorator(ak, options.SignModeHandler)
		if noSignatureVerification {
			sigVerificationDecorator = noopAnteDecorator{}
		}

		noTimeoutHeight := os.Getenv("DEVNET_NO_TIMEOUT_HEIGHT") == "true" ||
			os.Getenv("DEVNET_NO_TIMEOUT_HEIGHT") == "1"

		var txTimeoutHeightDecorator sdk.AnteDecorator = authante.NewTxTimeoutHeightDecorator()
		if noTimeoutHeight {
			txTimeoutHeightDecorator = noopAnteDecorator{}
		}

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
							// don't allow EVM messages in this route:
							evmante.RejectEthMessagesDecorator{},
							// disable the Msg types that cannot be included on an authz.MsgExec msgs field, e.g. EVM messages:
							NewAuthzLimiterDecorator(options.DisabledAuthzMsgs),

							authante.NewSetUpContextDecorator(),                                              // outermost AnteDecorator. SetUpContext must be called first
							wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
							wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
							authante.NewValidateBasicDecorator(),
							txTimeoutHeightDecorator,
							authante.NewValidateMemoDecorator(ak),
							authante.NewConsumeGasForTxSizeDecorator(ak),
							authante.NewSetPubKeyDecorator(ak), // SetPubKeyDecorator must be called before all signature verification decorators
							authante.NewValidateSigCountDecorator(ak),
							txfeeskeeper.NewMempoolFeeDecorator(options.TxFeesKeeper, false),
							NewDeductFeeDecorator(ak, options.BankKeeper), // overidden for fee delegation, also checks gas price against validator's config min gas prices during CheckTx
							authante.NewSigGasConsumeDecorator(ak, DefaultSigVerificationGasConsumer),
							eip712SigVerificationDecorator,             // overidden for EIP712 Tx signatures
							authante.NewIncrementSequenceDecorator(ak), // innermost AnteDecorator
						)
					default:
						return ctx, errors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
					}
				case "/injective.evm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler, err = newEVMAnteHandler(options)
					if err != nil {
						return ctx, errors.Wrapf(sdkerrors.ErrAppConfig, "can't create Eth Ante handler: %v", err)
					}
				default:
					ctx.Logger().Error("rejecting tx with unsupported extension option", "type_url", typeURL)
					return ctx, sdkerrors.ErrUnknownExtensionOptions
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx

		switch tx.(type) {
		case sdk.Tx:
			anteHandler = sdk.ChainAnteDecorators(
				// don't allow EVM messages in this route:
				evmante.RejectEthMessagesDecorator{},
				// disable the Msg types that cannot be included on an authz.MsgExec msgs field, e.g. EVM messages:
				NewAuthzLimiterDecorator(options.DisabledAuthzMsgs),

				authante.NewSetUpContextDecorator(),                                              // outermost AnteDecorator. SetUpContext must be called first
				wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
				wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
				authante.NewExtensionOptionsDecorator(nil),
				authante.NewValidateBasicDecorator(),
				txTimeoutHeightDecorator,
				authante.NewValidateMemoDecorator(ak),
				txfeeskeeper.NewMempoolFeeDecorator(options.TxFeesKeeper, false),
				authante.NewConsumeGasForTxSizeDecorator(ak),
				authante.NewDeductFeeDecorator(ak, options.BankKeeper, options.FeegrantKeeper, nil),
				authante.NewSetPubKeyDecorator(ak), // SetPubKeyDecorator must be called before all signature verification decorators
				authante.NewValidateSigCountDecorator(ak),
				authante.NewSigGasConsumeDecorator(ak, DefaultSigVerificationGasConsumer),
				sigVerificationDecorator,
				authante.NewIncrementSequenceDecorator(ak),
				ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
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
	meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params,
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
	meter storetypes.GasMeter, sig *signing.MultiSignatureData, pubkey multisig.PubKey,
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
