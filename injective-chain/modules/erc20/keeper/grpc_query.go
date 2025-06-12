package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{Keeper: k}
}

func (q queryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(sdk.UnwrapSDKContext(c))}, nil
}

func (q queryServer) AllTokenPairs(c context.Context, req *types.QueryAllTokenPairsRequest) (*types.QueryAllTokenPairsResponse, error) {
	if req == nil {
		return nil, errors.Wrap(types.ErrInvalidQueryRequest, "no request provided")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := q.getTokenPairsStoreByBankDenom(ctx)
	pairs := make([]*types.TokenPair, 0)

	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		pair := &types.TokenPair{
			BankDenom:    string(key),
			Erc20Address: common.BytesToAddress(value).String(),
		}
		pairs = append(pairs, pair)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "can't paginate request")
	}

	return &types.QueryAllTokenPairsResponse{
		TokenPairs: pairs,
		Pagination: pageRes,
	}, nil
}

func (q queryServer) TokenPairByDenom(c context.Context, req *types.QueryTokenPairByDenomRequest) (*types.QueryTokenPairByDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	pair, err := q.GetTokenPairForDenom(ctx, req.BankDenom)
	if err != nil {
		return nil, err
	}

	return &types.QueryTokenPairByDenomResponse{TokenPair: pair}, nil
}

func (q queryServer) TokenPairByERC20Address(c context.Context, req *types.QueryTokenPairByERC20AddressRequest) (*types.QueryTokenPairByERC20AddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	erc20Address := common.HexToAddress(req.Erc20Address)

	pair, err := q.GetTokenPairForERC20(ctx, erc20Address)
	if err != nil {
		return nil, err
	}

	return &types.QueryTokenPairByERC20AddressResponse{TokenPair: pair}, nil
}
