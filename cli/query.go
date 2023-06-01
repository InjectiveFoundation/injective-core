package cli

import (
	"context"
	"reflect"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli/flags"
)

// QueryCmd generates single query command
func QueryCmd[querier any](use, short string, newQueryClientFn func(grpc.ClientConn) querier, msg proto.Message, flagsMap FlagsMapping, argsMap ArgsMapping) *cobra.Command {
	numArgs := parseNumFields(msg, flagsMap, argsMap)

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(numArgs - len(flagsMap)),
		RunE:  queryRunE(newQueryClientFn, msg, flagsMap, argsMap),
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func queryRunE[querier any](newQueryClientFn func(grpc.ClientConn) querier, msg proto.Message, flagsMap FlagsMapping, argsMap ArgsMapping) func(*cobra.Command, []string) error {

	fnName := parseExpectedQueryFnName(msg)

	return func(cmd *cobra.Command, args []string) error {
		clientCtx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		queryClient := newQueryClientFn(clientCtx)

		err = parseFieldsFromFlagsAndArgs(msg, flagsMap, argsMap, cmd.Flags(), args, clientCtx)
		if err != nil {
			return err
		}

		res, err := callQueryClientFn(cmd.Context(), fnName, msg, queryClient)
		if err != nil {
			return err
		}

		return clientCtx.PrintProto(res)
	}
}

func callQueryClientFn(ctx context.Context, fnName string, msg proto.Message, queryClient any) (res proto.Message, err error) {
	qVal := reflect.ValueOf(queryClient)
	method := qVal.MethodByName(fnName)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(msg),
	}
	results := method.Call(args)
	if len(results) != 2 {
		panic("We got something wrong")
	}
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
		return res, err
	}
	res = results[0].Interface().(proto.Message)
	return res, nil
}
