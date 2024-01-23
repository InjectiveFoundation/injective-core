# Overview

Package `cli` offers a convenient wrappers for CLI command generation:
* module root command
* query
* tx

Both Query and Tx functions follow the same principle: for general cases, you only need to provide pointer to instance of the empty msg type struct which is to be sent to querier / broadcasted as txn. For cases when you need more freedom with how to parse input data, you can fill in `flagsMap` and `argsMap` mappings.

`flagsMap` is used to remap msg struct "FieldName" => cli FlagName\
`argsMap` is used to remap msg struct "FieldName" => cli Arg by Index

Both of these mappings also support passing a Transform function, which has a signature of `func(origV string, ctx grpc.ClientConn) (tranformedV any, err error)` and is used when the input value needs additional handling before parsing into struct field.

Additionally, flags have `UseDefaultIfOmitted` param which alter the behaviour when the flag was not provided by the user. If `false`, the flag will be completely skipped. Otherwise we will substitute it with default value.

## How it works

1. It tries to fill the provided msg fields with inputs from cli arguments or flags:
	* If there is a defined mapping for the struct field name in either of `flagsMap` or `argsMap`, we get the value for this field from the mapping.
	* If the field is supposed to be filled from cli context (`--from` flag), like `Sender`, `SubaccountId`, etc., then fill the value from the context
	* If the field is not empty (you can pre-fill any of the msg fields), skip it
	* Otherwise, try to fill in the value from the next unconsumed argument
2. For `QueryCmd`, it deducts the appropriate function name on the querier (based on msg type name) and returns the result
3. For `TxCmd`, it broacasts the msg to the network.

To mark any of the msg fields as intentionally empty, pass that field into `flagsMap` with the `cli.SkipField` as mapping.

For complex messages that have some fields encoded as `proto.Any`, such as proposals, you need to hint the parser the internal struct type for that field by filling it in with the empty internal struct, as shown in the [Tx example](#example-with-custom-field-parsing)

## Queries

```go
QueryCmd[querier any](
	use string,
	short string,
	newQueryClientFn func(grpc.ClientConn) querier,
	msg proto.Message,
	flagsMap FlagsMapping,
	argsMap ArgsMapping
)
```

#### Simple Example

```go
func GetAuctionInfo() *cobra.Command {
	cmd := cli.QueryCmd(
		"info",
		"Gets current auction round info",
		types.NewQueryClient,
		&types.QueryCurrentAuctionBasketRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{})
	cmd.Long = "Gets current auction round info, including coin basket and highest bidder"
	return cmd
}
```

## Transactions

```go
TxCmd(
	use string,
	short string,
	msg sdk.Msg,
	flagsMap FlagsMapping,
	argsMap ArgsMapping
)
```

#### Simple Example

```go
func NewInstantSpotMarketLaunchTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"instant-spot-market-launch <ticker> <base_denom> <quote_denom>",
		"Launch spot market by paying listing fee without governance",
		&types.MsgInstantSpotMarketLaunch{},
		cli.FlagsMapping{"MinPriceTickSize": cli.Flag{Flag: FlagMinPriceTickSize, UseDefaultIfOmitted: true}, "MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize, UseDefaultIfOmitted: true}},
		cli.ArgsMapping{},
	)
	cmd.Example = `tx exchange instant-spot-market-launch INJ/ATOM uinj uatom --min-price-tick-size=1000000000 --min-quantity-tick-size=1000000000000000`
	cmd.Flags().String(FlagMinPriceTickSize, "1000000000", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "1000000000000000", "min quantity tick size")
	return cmd
}
```

#### Example with custom field parsing

```go
func NewSpotMarketUpdateParamsProposalTxCmd() *cobra.Command {
	proposalMsgDummy := &govtypes.MsgSubmitProposal{}
	_ = proposalMsgDummy.SetContent(&types.SpotMarketParamUpdateProposal{})
	cmd := cli.TxCmd(
		"update-spot-market-params",
		"Submit a proposal to update spot market params",
		proposalMsgDummy, cli.FlagsMapping{
			"Title":               cli.Flag{Flag: govcli.FlagTitle},
			"Description":         cli.Flag{Flag: govcli.FlagDescription},
			"MarketId":            cli.Flag{Flag: FlagMarketID},
			"MakerFeeRate":        cli.Flag{Flag: FlagMakerFeeRate},
			"TakerFeeRate":        cli.Flag{Flag: FlagTakerFeeRate},
			"RelayerFeeShareRate": cli.Flag{Flag: FlagRelayerFeeShareRate},
			"MinPriceTickSize":    cli.Flag{Flag: FlagMinPriceTickSize},
			"MinQuantityTickSize": cli.Flag{Flag: FlagMinQuantityTickSize},
			"Status": cli.Flag{Flag: FlagMarketStatus, Transform: func(origV string, ctx grpc.ClientConn) (tranformedV any, err error) {
				var status types.MarketStatus
				if origV != "" {
					if newStatus, ok := types.MarketStatus_value[origV]; ok {
						status = types.MarketStatus(newStatus)
					} else {
						return nil, fmt.Errorf("incorrect market status: %s", origV)
					}
				} else {
					status = types.MarketStatus_Unspecified
				}
				return fmt.Sprintf("%v", int32(status)), nil
			}},
			"InitialDeposit": cli.Flag{Flag: govcli.FlagDeposit},
		}, cli.ArgsMapping{})
	cmd.Example = `tx exchange update-spot-market-params --market-id="0xacdd4f9cb90ecf5c4e254acbf65a942f562ca33ba718737a93e5cb3caadec3aa" --title="Spot market params update" --description="XX" --deposit="1000000000000000000inj"`

	cmd.Flags().String(FlagMarketID, "", "Spot market ID")
	cmd.Flags().String(FlagMakerFeeRate, "", "maker fee rate")
	cmd.Flags().String(FlagTakerFeeRate, "", "taker fee rate")
	cmd.Flags().String(FlagRelayerFeeShareRate, "", "relayer fee share rate")
	cmd.Flags().String(FlagMinPriceTickSize, "", "min price tick size")
	cmd.Flags().String(FlagMinQuantityTickSize, "", "min quantity tick size")
	cmd.Flags().String(FlagMarketStatus, "", "market status")
	cliflags.AddGovProposalFlags(cmd)

	return cmd
}
```

## Root module command

```go
ModuleRootCommand(moduleName string, isQuery bool)
```

#### Example

```go
cmd := cli.ModuleRootCommand(types.ModuleName, false)
```