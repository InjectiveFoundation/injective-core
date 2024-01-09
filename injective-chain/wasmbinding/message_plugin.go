package wasmbinding

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"

	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding/bindings"
)

// CustomMessageDecorator returns decorator for custom CosmWasm bindings messages
func CustomMessageDecorator(
	router wasmkeeper.MessageRouter,
	bankKeeper bankkeeper.BaseKeeper,
	exchangeKeeper *exchangekeeper.Keeper,
	tokenFactoryKeeper *tokenfactorykeeper.Keeper,
) func(wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(old wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomMessenger{
			router:             router,
			wrapped:            old,
			bankKeeper:         &bankKeeper,
			exchangeKeeper:     exchangeKeeper,
			tokenFactoryKeeper: tokenFactoryKeeper,
		}
	}
}

type CustomMessenger struct {
	router             wasmkeeper.MessageRouter
	wrapped            wasmkeeper.Messenger
	bankKeeper         *bankkeeper.BaseKeeper
	exchangeKeeper     *exchangekeeper.Keeper
	tokenFactoryKeeper *tokenfactorykeeper.Keeper
}

func (m *CustomMessenger) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
	if msg.Custom == nil {
		return m.wrapped.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	}

	var wrappedContractMsg bindings.InjectiveMsgWrapper
	if err := json.Unmarshal(msg.Custom, &wrappedContractMsg); err != nil {
		return nil, nil, errors.Wrap(err, "Error parsing msg data")
	}

	var contractMsg bindings.InjectiveMsg

	if err := json.Unmarshal(wrappedContractMsg.MsgData, &contractMsg); err != nil {
		return nil, nil, errors.Wrap(err, "injective msg")
	}

	var sdkMsg sdk.Msg

	switch {
	/// tokenfactory msgs
	case contractMsg.CreateDenom != nil:
		sdkMsg = contractMsg.CreateDenom
	case contractMsg.ChangeAdmin != nil:
		sdkMsg = contractMsg.ChangeAdmin
	case contractMsg.MintTokens != nil:
		// special case: since MsgMint's handler doesn't allow sending directly to recipient, but we want to expose this
		mint := contractMsg.MintTokens

		rcpt, err := parseAddress(mint.MintTo)
		if err != nil {
			return nil, nil, err
		}

		sdkMsg = tokenfactorytypes.NewMsgMint(contractAddr.String(), mint.Amount)

		events, data, err := m.handleSdkMessageWithResults(ctx, contractAddr, sdkMsg)
		if err != nil {
			return nil, nil, err
		}

		// create context with new event manager
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		err = m.bankKeeper.SendCoins(ctx, contractAddr, rcpt, sdk.NewCoins(mint.Amount))
		if err != nil {
			return nil, nil, errors.Wrap(err, "sending newly minted coins from message")
		}
		// merge the events together
		events = append(events, ctx.EventManager().Events()...)
		return events, data, nil
	case contractMsg.BurnTokens != nil:
		contractMsg.BurnTokens.Sender = contractAddr.String()
		sdkMsg = contractMsg.BurnTokens
	case contractMsg.SetTokenMetadata != nil:
		mt := contractMsg.SetTokenMetadata
		sdkMsg = &tokenfactorytypes.MsgSetDenomMetadata{
			Sender: contractAddr.String(),
			Metadata: banktypes.Metadata{
				Base:    mt.Denom,
				Display: mt.Symbol,
				Name:    mt.Name,
				Symbol:  mt.Symbol,
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    mt.Denom,
						Exponent: 0,
					},
					{
						Denom:    mt.Symbol,
						Exponent: mt.Decimals,
					},
				},
			},
		}
	/// oracle msgs
	case contractMsg.RelayPythPrices != nil:
		sdkMsg = contractMsg.RelayPythPrices
	/// exchange msgs
	case contractMsg.BatchUpdateOrders != nil:
		sdkMsg = contractMsg.BatchUpdateOrders
	case contractMsg.PrivilegedExecuteContract != nil:
		sdkMsg = contractMsg.PrivilegedExecuteContract
	case contractMsg.Deposit != nil:
		sdkMsg = contractMsg.Deposit
	case contractMsg.Withdraw != nil:
		sdkMsg = contractMsg.Withdraw
	case contractMsg.CreateSpotLimitOrder != nil:
		sdkMsg = contractMsg.CreateSpotLimitOrder
	case contractMsg.BatchCreateSpotLimitOrders != nil:
		sdkMsg = contractMsg.BatchCreateSpotLimitOrders
	case contractMsg.CreateSpotMarketOrder != nil:
		sdkMsg = contractMsg.CreateSpotMarketOrder
	case contractMsg.CancelSpotOrder != nil:
		sdkMsg = contractMsg.CancelSpotOrder
	case contractMsg.BatchCancelSpotOrders != nil:
		sdkMsg = contractMsg.BatchCancelSpotOrders
	case contractMsg.CreateDerivativeLimitOrder != nil:
		sdkMsg = contractMsg.CreateDerivativeLimitOrder
	case contractMsg.BatchCreateDerivativeLimitOrders != nil:
		sdkMsg = contractMsg.BatchCreateDerivativeLimitOrders
	case contractMsg.CreateDerivativeMarketOrder != nil:
		sdkMsg = contractMsg.CreateDerivativeMarketOrder
	case contractMsg.CancelDerivativeOrder != nil:
		sdkMsg = contractMsg.CancelDerivativeOrder
	case contractMsg.BatchCancelDerivativeOrders != nil:
		sdkMsg = contractMsg.BatchCancelDerivativeOrders
	case contractMsg.SubaccountTransfer != nil:
		sdkMsg = contractMsg.SubaccountTransfer
	case contractMsg.ExternalTransfer != nil:
		sdkMsg = contractMsg.ExternalTransfer
	case contractMsg.IncreasePositionMargin != nil:
		sdkMsg = contractMsg.IncreasePositionMargin
	case contractMsg.LiquidatePosition != nil:
		sdkMsg = contractMsg.LiquidatePosition
	case contractMsg.InstantSpotMarketLaunch != nil:
		sdkMsg = contractMsg.InstantSpotMarketLaunch
	case contractMsg.InstantPerpetualMarketLaunch != nil:
		sdkMsg = contractMsg.InstantPerpetualMarketLaunch
	case contractMsg.InstantExpiryFuturesMarketLaunch != nil:
		sdkMsg = contractMsg.InstantExpiryFuturesMarketLaunch
	// wasmx messages
	case contractMsg.UpdateContractMsg != nil:
		sdkMsg = contractMsg.UpdateContractMsg
	case contractMsg.DeactivateContractMsg != nil:
		sdkMsg = contractMsg.DeactivateContractMsg
	case contractMsg.ActivateContractMsg != nil:
		sdkMsg = contractMsg.ActivateContractMsg
	case contractMsg.RewardsOptOut != nil:
		sdkMsg = contractMsg.RewardsOptOut
	default:
		return nil, nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("Unknown Injective Wasm Message: %T", contractMsg)}
	}

	return m.handleSdkMessageWithResults(ctx, contractAddr, sdkMsg)
}

func (m *CustomMessenger) handleSdkMessageWithResults(
	ctx sdk.Context,
	contractAddr sdk.Address,
	msg sdk.Msg,
) (events []sdk.Event, data [][]byte, err error) {
	res, err := m.handleSdkMessage(ctx, contractAddr, msg)
	if err != nil {
		return nil, nil, err
	}

	// append data
	data = append(data, res.Data)
	// append events
	sdkEvents := make([]sdk.Event, len(res.Events))
	for i := range res.Events {
		sdkEvents[i] = sdk.Event(res.Events[i])
	}
	events = append(events, sdkEvents...)
	return events, data, nil
}

// This function is forked from wasmd. sdk.Msg will be validated and routed to the corresponding module msg server in this function.
func (m *CustomMessenger) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) (*sdk.Result, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	// make sure this account can send it
	for _, acct := range msg.GetSigners() {
		if !acct.Equals(contractAddr) {
			return nil, errors.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
		}
	}

	// find the handler and execute it
	if handler := m.router.Handler(msg); handler != nil {
		// ADR 031 request type routing
		msgResult, err := handler(ctx, msg)
		return msgResult, err
	}
	// legacy sdk.Msg routing
	// Assuming that the app developer has migrated all their Msgs to
	// proto messages and has registered all `Msg services`, then this
	// path should never be called, because all those Msgs should be
	// registered within the `msgServiceRouter` already.
	return nil, errors.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
}

// parseAddress parses address from bech32 string and verifies its format.
func parseAddress(addr string) (sdk.AccAddress, error) {
	parsed, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return nil, errors.Wrap(err, "address from bech32")
	}
	err = sdk.VerifyAddressFormat(parsed)
	if err != nil {
		return nil, errors.Wrap(err, "verify address format")
	}
	return parsed, nil
}
