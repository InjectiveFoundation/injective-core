package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnpackInterfaces implements UnpackInterfacesMesssage.UnpackInterfaces
func (m QueryTraceTxRequest) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for i, msg := range m.Predecessors {
		if msg == nil {
			return status.Errorf(codes.InvalidArgument, "predecessors[%d]: nil entry", i)
		}
		if err := msg.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	if m.Msg == nil {
		return status.Error(codes.InvalidArgument, "empty request")
	}
	return m.Msg.UnpackInterfaces(unpacker)
}

func (m QueryTraceBlockRequest) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	for _, msg := range m.Txs {
		if err := msg.UnpackInterfaces(unpacker); err != nil {
			return err
		}
	}
	return nil
}
