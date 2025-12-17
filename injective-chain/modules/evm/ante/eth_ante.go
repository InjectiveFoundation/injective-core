package ante

import (
	"bytes"
	"errors"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/params"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/statedb"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	txfeeskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// EthAnteHandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper, EVM Keeper and Fee Market Keeper.
type EthAnteHandlerOptions struct {
	AccountKeeper          evmtypes.AccountKeeper
	BankKeeper             evmtypes.BankKeeper
	EvmKeeper              EVMKeeper
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	MaxTxGasWanted         uint64
	TxFeesDecorator        txfeeskeeper.MempoolFeeDecorator
	ExtensionOptionChecker ante.ExtensionOptionChecker
	DisabledAuthzMsgs      []string
	ExtraDecorators        []sdk.AnteDecorator
}

func (options EthAnteHandlerOptions) validate() error {
	if options.AccountKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "sign mode handler is required for ante builder")
	}
	if options.EvmKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "evm keeper is required for AnteHandler")
	}
	if options.TxFeesDecorator.TxFeesKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "TxFeesDecorator is required for AnteHandler")
	}
	return nil
}

func NewEthAnteHandler(options EthAnteHandlerOptions) (sdk.AnteHandler, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		blockCfg, err := options.EvmKeeper.EVMBlockConfig(ctx)
		if err != nil {
			return ctx, errorsmod.Wrap(errortypes.ErrLogic, err.Error())
		}
		evmParams := &blockCfg.Params
		evmDenom := evmParams.EvmDenom
		rules := blockCfg.Rules
		ethSigner := ethtypes.MakeSigner(blockCfg.ChainConfig, blockCfg.BlockNumber, blockCfg.BlockTime)

		// all transactions must implement FeeTx
		_, ok := tx.(sdk.FeeTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrInvalidType, "invalid transaction type %T, expected sdk.FeeTx", tx)
		}

		// We need to setup an empty gas config so that the gas is consistent with Ethereum.
		ctx, err = SetupEthContext(ctx)
		if err != nil {
			return ctx, err
		}

		// fallback ante handler that will be called after TxFeesDecorator.AnteHandle() succeeds
		// this ante handle will only be active if Mempool1559Enabled is not enabled, aka no dynamic fees
		// so we simply check that gas price is >= minGasPrice of the validator
		nextAnteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			txFeesParams := options.TxFeesDecorator.TxFeesKeeper.GetParams(ctx)
			if txFeesParams.Mempool1559Enabled { // we checked against dynamic txfees
				return ctx, nil
			}
			return ctx, CheckEthMempoolFee(ctx, tx, simulate, evmDenom) // check against validator static min gas price
		}
		// NB: this ante handler only checks gas price of the whole cosmos tx, and not the gas price of individual MsgEthereumTx inside cosmos tx,
		// but this is okey for us due to multiple reasons:
		// 1) we check that sum of each msg's fee and gas limit equal to tx fee and gas limit inside ValidateEthBasic ante
		// 2) we do not refund unused gas.
		// 3) whole tx is atomic, no partial execution possible
		// This means users can set gas prices inside individual msgs lower than required minimum, but the total gas price
		// should be >= current base fee (so some other msgs should offset for that by setting gas prices higher than required)
		if _, err := options.TxFeesDecorator.AnteHandle(ctx, tx, simulate, nextAnteHandler); err != nil {
			return ctx, err
		}

		if err := ValidateEthBasic(ctx, tx, evmParams); err != nil {
			return ctx, err
		}

		if err := VerifyEthSig(tx, ethSigner); err != nil {
			return ctx, err
		}

		if err := VerifyEthAccount(ctx, tx, options.EvmKeeper, options.AccountKeeper, evmDenom); err != nil {
			return ctx, err
		}

		if err := CheckEthCanTransfer(ctx, tx, rules, options.EvmKeeper, evmParams); err != nil {
			return ctx, err
		}

		ctx, err = CheckEthGasConsume(
			ctx, tx, rules, options.EvmKeeper,
			options.MaxTxGasWanted, evmDenom,
		)
		if err != nil {
			return ctx, err
		}

		if err := CheckEthSenderNonce(ctx, tx, options.AccountKeeper); err != nil {
			return ctx, err
		}

		extraDecorators := options.ExtraDecorators
		if len(extraDecorators) > 0 {
			return sdk.ChainAnteDecorators(extraDecorators...)(ctx, tx, simulate)
		}

		return ctx, nil
	}, nil
}

// TODO: (xlab) entire file has to be refactored to the point where all "checks" and state mutations
// are povided by the evm module and here just plugged into the sdk.AnteHandler.

// EVMKeeper defines the expected keeper interface used on the Eth AnteHandler
type EVMKeeper interface {
	statedb.Keeper
	EIP155ChainID(sdk.Context) *big.Int
	EVMBlockConfig(sdk.Context) (*evmkeeper.EVMBlockConfig, error)
	DeductTxCostsFromUserBalance(ctx sdk.Context, fees sdk.Coins, from common.Address) error
}

type protoTxProvider interface {
	GetProtoTx() *sdktx.Tx
}

// SetupEthContext is adapted from SetUpContextDecorator from cosmos-sdk, it ignores gas consumption
// by setting the gas meter to infinite
func SetupEthContext(ctx sdk.Context) (newCtx sdk.Context, err error) {
	// We need to setup an empty gas config so that the gas is consistent with Ethereum.
	newCtx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})

	return newCtx, nil
}

// ValidateEthBasic handles basic validation of tx
func ValidateEthBasic(ctx sdk.Context, tx sdk.Tx, evmParams *evmtypes.Params) error {
	// no need to validate basic on recheck tx, call next antehandler
	if ctx.IsReCheckTx() {
		return nil
	}

	msgs := tx.GetMsgs()
	if msgs == nil {
		return errorsmod.Wrap(errortypes.ErrUnknownRequest, "invalid transaction. Transaction without messages")
	}

	if t, ok := tx.(sdk.HasValidateBasic); ok {
		err := t.ValidateBasic()
		// ErrNoSignatures is fine with eth tx
		if err != nil && !errors.Is(err, errortypes.ErrNoSignatures) {
			return errorsmod.Wrap(err, "tx basic validation failed")
		}
	}

	// For eth type cosmos tx, some fields should be verified as zero values,
	// since we will only verify the signature against the hash of the MsgEthereumTx.Data
	wrapperTx, ok := tx.(protoTxProvider)
	if !ok {
		return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid tx type %T, didn't implement interface protoTxProvider", tx)
	}

	protoTx := wrapperTx.GetProtoTx()
	body := protoTx.Body
	if body.Memo != "" || body.TimeoutHeight != uint64(0) || len(body.NonCriticalExtensionOptions) > 0 {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest,
			"for eth tx body Memo TimeoutHeight NonCriticalExtensionOptions should be empty")
	}

	if len(body.ExtensionOptions) != 1 {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx length of ExtensionOptions should be 1")
	}

	authInfo := protoTx.AuthInfo
	if len(authInfo.SignerInfos) > 0 {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx AuthInfo SignerInfos should be empty")
	}

	if authInfo.Tip != nil {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx AuthInfo Tip should be empty")
	}

	if authInfo.Fee.Payer != "" || authInfo.Fee.Granter != "" {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx AuthInfo Fee payer and granter should be empty")
	}

	sigs := protoTx.Signatures
	if len(sigs) > 0 {
		return errorsmod.Wrap(errortypes.ErrInvalidRequest, "for eth tx Signatures should be empty")
	}

	txFee := sdk.Coins{}
	txGasLimit := uint64(0)

	enableCreate := evmParams.GetEnableCreate()
	enableCall := evmParams.GetEnableCall()
	evmDenom := evmParams.GetEvmDenom()
	allowUnprotectedTxs := evmParams.GetAllowUnprotectedTxs()

	for _, msg := range protoTx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		txGasLimit += msgEthTx.GetGas()

		tx := msgEthTx.AsTransaction()
		// return error if contract creation or call are disabled through governance
		if !enableCreate && tx.To() == nil {
			return errorsmod.Wrap(evmtypes.ErrCreateDisabled, "failed to create new contract")
		} else if !enableCall && tx.To() != nil {
			return errorsmod.Wrap(evmtypes.ErrCallDisabled, "failed to call contract")
		}

		if !allowUnprotectedTxs && !tx.Protected() {
			return errorsmod.Wrapf(
				errortypes.ErrNotSupported,
				"rejected unprotected Ethereum transaction. Please EIP155 sign your transaction to protect it against replay-attacks")
		}

		txFee = txFee.Add(sdk.Coin{Denom: evmDenom, Amount: sdkmath.NewIntFromBigInt(msgEthTx.GetFee())})
	}

	if !authInfo.Fee.Amount.Equal(txFee) {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid AuthInfo Fee Amount (%s != %s)", authInfo.Fee.Amount, txFee)
	}

	if authInfo.Fee.GasLimit != txGasLimit {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid AuthInfo Fee GasLimit (%d != %d)", authInfo.Fee.GasLimit, txGasLimit)
	}

	return nil
}

// RejectEthMessagesDecorator prevents MsgEthereumTx msg types from being executed directly with no extension options.
type RejectEthMessagesDecorator struct{}

// AnteHandle will reject messages that requires ethereum-specific authentication.
func (rmd RejectEthMessagesDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		if _, ok := msg.(*evmtypes.MsgEthereumTx); ok {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrInvalidType,
				"MsgEthereumTx needs to be contained within a tx with 'ExtensionOptionsEthereumTx' option",
			)
		}
	}

	return next(ctx, tx, simulate)
}

// VerifyEthSig validates checks that the registered chain id is the same as the one on the message, and
// that the signer address matches the one defined on the message.
// It's not skipped for RecheckTx, because it set `From` address which is critical from other ante handler to work.
// Failure in RecheckTx will prevent tx to be included into block, especially when CheckTx succeed, in which case user
// won't see the error message.
func VerifyEthSig(tx sdk.Tx, signer ethtypes.Signer) error {
	var firstMsgSender []byte

	for i, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		if err := msgEthTx.VerifySender(signer); err != nil {
			return errorsmod.Wrapf(errortypes.ErrorInvalidSigner, "signature verification failed: %s", err.Error())
		}

		// ensure that all the msgs in a tx are signed by the same sender: https://bugs.immunefi.com/magnus/1515/projects/383/bug-bounty/reports/58062
		if i == 0 {
			firstMsgSender = msgEthTx.From
		} else if !bytes.Equal(firstMsgSender, msgEthTx.From) {
			return errorsmod.Wrap(errortypes.ErrorInvalidSigner, "not all msgs are signed by the same signer")
		}
	}

	return nil
}

// CheckEthMempoolFee will check if the transaction's effective fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator
//
// AnteHandle ensures that the provided fees meet a minimum threshold for the validator.
// This check only for local mempool purposes, and thus it is only run on (Re)CheckTx.
// The logic is also skipped if the London hard fork and EIP-1559 are enabled.
func CheckEthMempoolFee(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	evmDenom string,
) error {
	if !ctx.IsCheckTx() || simulate {
		return nil
	}

	minGasPrice := ctx.MinGasPrices().AmountOf(evmDenom)

	for _, msg := range tx.GetMsgs() {
		ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		fee := sdkmath.LegacyNewDecFromBigInt(ethMsg.GetFee())
		gasLimit := sdkmath.LegacyNewDecFromBigInt(new(big.Int).SetUint64(ethMsg.GetGas()))
		requiredFee := minGasPrice.Mul(gasLimit)

		if fee.LT(requiredFee) {
			return errorsmod.Wrapf(
				errortypes.ErrInsufficientFee,
				"insufficient fee; got: %s required: %s",
				fee, requiredFee,
			)
		}
	}

	return nil
}

// VerifyEthAccount validates checks that the sender balance is greater than the total transaction cost.
// The account will be set to store if it doesn't exis, i.e cannot be found on store.
// This AnteHandler decorator will fail if:
// - any of the msgs is not a MsgEthereumTx
// - from address is empty
// - account balance is lower than the transaction cost
func VerifyEthAccount(
	ctx sdk.Context, tx sdk.Tx,
	evmKeeper EVMKeeper, ak evmtypes.AccountKeeper, evmDenom string,
) error {
	if !ctx.IsCheckTx() {
		return nil
	}

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		ethTx := msgEthTx.AsTransaction()

		// sender address should be in the tx cache from the previous AnteHandle call
		from := msgEthTx.GetFrom()
		if from.Empty() {
			return errorsmod.Wrap(errortypes.ErrInvalidAddress, "from address cannot be empty")
		}

		// check whether the sender address is EOA
		fromAddr := common.BytesToAddress(from)
		acct := evmKeeper.GetAccount(ctx, fromAddr)

		if acct == nil {
			acc := ak.NewAccountWithAddress(ctx, from)
			ak.SetAccount(ctx, acc)
		} else if acct.IsContract() {
			return errorsmod.Wrapf(errortypes.ErrInvalidType,
				"the sender is not EOA: address %s, codeHash <%s>", fromAddr, acct.CodeHash)
		}

		balance := evmKeeper.GetBalance(ctx, from, evmDenom)
		if err := evmkeeper.CheckSenderBalance(sdkmath.NewIntFromBigIntMut(balance), ethTx); err != nil {
			return errorsmod.Wrap(err, "failed to check sender balance")
		}
	}
	return nil
}

// CheckEthGasConsume validates that the Ethereum tx message has enough to cover intrinsic gas
// (during CheckTx only) and that the sender has enough balance to pay for the gas cost.
//
// Intrinsic gas for a transaction is the amount of gas that the transaction uses before the
// transaction is executed. The gas is a constant value plus any cost incurred by additional bytes
// of data supplied with the transaction.
//
// This AnteHandler decorator will fail if:
// - the message is not a MsgEthereumTx
// - sender account cannot be found
// - transaction's gas limit is lower than the intrinsic gas
// - user doesn't have enough balance to deduct the transaction fees (gas_limit * gas_price)
// - transaction or block gas meter runs out of gas
// - sets the gas meter limit
// - gas limit is greater than the block gas meter limit
func CheckEthGasConsume(
	ctx sdk.Context, tx sdk.Tx,
	rules params.Rules,
	evmKeeper EVMKeeper,
	maxGasWanted uint64,
	evmDenom string,
) (sdk.Context, error) {
	gasWanted := uint64(0)

	// safeAddGas is a helper function to add gas to the gas wanted and returns
	// an error if the gas wanted overflows the maximum uint64 value.
	safeAddGas := func(gas uint64) error {
		if gas > math.MaxUint64-gasWanted {
			return errorsmod.Wrapf(
				errortypes.ErrOutOfGas, "gas wanted overflow: %d + %d > %v", gasWanted, gas, uint64(math.MaxUint64))
		}
		gasWanted += gas
		return nil
	}

	var events sdk.Events

	// Use the lowest priority of all the messages as the final one.
	minPriority := int64(math.MaxInt64)

	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		priority := evmtypes.GetTxPriority(msgEthTx)

		if priority < minPriority {
			minPriority = priority
		}

		gasLimit := msgEthTx.GetGas()
		if ctx.IsCheckTx() && maxGasWanted != 0 {
			// We can't trust the tx gas limit, because we'll refund the unused gas.
			if gasLimit > maxGasWanted {
				if err := safeAddGas(maxGasWanted); err != nil {
					return ctx, err
				}
			} else {
				if err := safeAddGas(gasLimit); err != nil {
					return ctx, err
				}
			}
		} else {
			if err := safeAddGas(gasLimit); err != nil {
				return ctx, err
			}
		}

		// user balance is already checked during CheckTx so there's no need to
		// verify it again during ReCheckTx
		if ctx.IsReCheckTx() {
			continue
		}

		fees, err := evmkeeper.VerifyFee(msgEthTx, evmDenom, rules.IsHomestead, rules.IsIstanbul, rules.IsShanghai, ctx.IsCheckTx())
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "failed to verify the fees")
		}

		err = evmKeeper.DeductTxCostsFromUserBalance(ctx, fees, common.BytesToAddress(msgEthTx.From))
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "failed to deduct transaction costs from user balance")
		}

		events = append(events,
			sdk.NewEvent(
				sdk.EventTypeTx,
				sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
			),
		)
	}

	ctx.EventManager().EmitEvents(events)

	blockGasLimit := chaintypes.BlockGasLimit(ctx)

	// return error if the tx gas is greater than the block limit (max gas)

	// NOTE: it's important here to use the gas wanted instead of the gas consumed
	// from the tx gas pool. The later only has the value so far since the
	// EthSetupContextDecorator so it will never exceed the block gas limit.
	if gasWanted > blockGasLimit {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrOutOfGas,
			"tx gas (%d) exceeds block gas limit (%d)",
			gasWanted,
			blockGasLimit,
		)
	}

	// Set tx GasMeter with a limit of GasWanted (i.e gas limit from the Ethereum tx).
	// The gas consumed will be then reset to the gas used by the state transition
	// in the EVM.

	// FIXME: use a custom gas configuration that doesn't add any additional gas and only
	// takes into account the gas consumed at the end of the EVM transaction.
	newCtx := ctx.
		WithGasMeter(chaintypes.NewThreadsafeInfiniteGasMeterWithLimit(gasWanted)).
		WithPriority(minPriority)

	// we know that we have enough gas on the pool to cover the intrinsic gas
	return newCtx, nil
}

// CheckEthCanTransfer creates an EVM from the message and calls the BlockContext CanTransfer function to
// see if the address can execute the transaction.
func CheckEthCanTransfer(
	ctx sdk.Context,
	tx sdk.Tx,
	rules params.Rules,
	evmKeeper EVMKeeper,
	evmParams *evmtypes.Params,
) error {
	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		tx := msgEthTx.AsTransaction()

		from := common.BytesToAddress(msgEthTx.From)
		// check that caller has enough balance to cover asset transfer for **topmost** call
		// NOTE: here the gas consumed is from the context with the infinite gas meter
		if tx.Value().Sign() > 0 && !canTransfer(ctx, evmKeeper, evmParams.EvmDenom, from, tx.Value()) {
			return errorsmod.Wrapf(
				errortypes.ErrInsufficientFunds,
				"failed to transfer %s from address %s using the EVM block context transfer function",
				tx.Value(),
				from,
			)
		}
	}

	return nil
}

// canTransfer adapted the core.CanTransfer from go-ethereum
func canTransfer(ctx sdk.Context, evmKeeper EVMKeeper, denom string, from common.Address, amount *big.Int) bool {
	balance := evmKeeper.GetBalance(ctx, sdk.AccAddress(from.Bytes()), denom)
	return balance.Cmp(amount) >= 0
}

// CheckEthSenderNonce handles incrementing the sequence of the signer (i.e sender). If the transaction is a
// contract creation, the nonce will be incremented during the transaction execution and not within
// this AnteHandler decorator.
func CheckEthSenderNonce(
	ctx sdk.Context, tx sdk.Tx, ak evmtypes.AccountKeeper,
) error {
	for _, msg := range tx.GetMsgs() {
		msgEthTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid message type %T, expected %T", msg, (*evmtypes.MsgEthereumTx)(nil))
		}

		tx := msgEthTx.AsTransaction()

		// increase sequence of sender
		acc := ak.GetAccount(ctx, msgEthTx.GetFrom())
		if acc == nil {
			return errorsmod.Wrapf(
				errortypes.ErrUnknownAddress,
				"account %s is nil", common.BytesToAddress(msgEthTx.GetFrom().Bytes()),
			)
		}
		nonce := acc.GetSequence()

		// we merged the nonce verification to nonce increment, so when tx includes multiple messages
		// with same sender, they'll be accepted.
		if tx.Nonce() != nonce {
			return errorsmod.Wrapf(
				errortypes.ErrInvalidSequence,
				"invalid nonce; got %d, expected %d", tx.Nonce(), nonce,
			)
		}

		if err := acc.SetSequence(nonce + 1); err != nil {
			return errorsmod.Wrapf(err, "failed to set sequence to %d", acc.GetSequence()+1)
		}

		ak.SetAccount(ctx, acc)
	}

	return nil
}
