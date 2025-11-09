package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/pflag"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// parseNumFields returns number of zero fields in the struct that needs to be parsed from args
func parseNumFields(message any, flagsMap FlagsMapping, argsMap ArgsMapping) int {
	v := reflect.ValueOf(message).Elem()
	num := v.Type().NumField()
	for i := 0; i < v.Type().NumField(); i++ {
		field := v.Field(i)
		fieldT := field.Type()
		fName := v.Type().Field(i).Name
		switch {
		case isFilledFromCtx(fName): // parsed from context "from"
			if _, ok := flagsMap[fName]; ok {
				continue
			}
			if _, ok := argsMap[fName]; ok {
				continue
			}
			num--
		case fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && fieldT.Elem().String() == "types.Any": // proto-encoded type, never nil
			concreteStructInterface := field.Elem().Interface()
			concreteStruct, ok := concreteStructInterface.(codectypes.Any)
			if !ok {
				continue
			}
			num += parseNumFields(concreteStruct.GetCachedValue(), flagsMap, argsMap)
			num-- // remove this field itself
		// recursively look for internal structs with empty fields
		case fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && !isComplexValue(fieldT.Elem().String()): // pointer to struct
			num += parseNumFields(reflect.New(fieldT.Elem()).Interface(), flagsMap, argsMap)
			num-- // remove this field itself
		case fieldT.Kind() == reflect.Struct && !isComplexValue(fieldT.String()): // struct
			num += parseNumFields(reflect.New(fieldT).Interface(), flagsMap, argsMap)
			num-- // remove this field itself
		case !field.IsZero() && !isZeroNumber(field): // skip filled fields
			num--
		}
	}
	return num
}

// fillSenderFromCtx fills in "Sender", "FeeRecipient", "Proposer" and "SubaccountId" fields of msg and it's internal structs (if present) with the From value from the context
func fillSenderFromCtx(msg sdk.Msg, clientCtx client.Context) error {
	fromAddress := clientCtx.GetFromAddress().String()
	var fillSenderInStruct func(any) error
	fillSenderInStruct = func(msg any) error {
		v := reflect.ValueOf(msg).Elem()
		t := v.Type()
		for i := 0; i < v.Type().NumField(); i++ {
			field := v.Field(i)
			fieldT := field.Type()
			fieldName := t.Field(i).Name
			switch {
			case isFilledFromCtx(fieldName) && field.IsZero():
				switch fieldName {
				case "SubaccountId", "SourceSubaccountId": // subaccounts
					senderAddr, err := sdk.AccAddressFromBech32(fromAddress)
					if err != nil {
						continue
					}
					ethAddress := common.BytesToAddress(senderAddr.Bytes())
					subaccountID := types.EthAddressToSubaccountID(ethAddress)
					field.SetString(subaccountID.Hex())
				default: // addresses
					field.SetString(fromAddress)
				}
			case fieldName == "Sender", fieldName == "FeeRecipient", fieldName == "Proposer": // parsed from context "from"
				if field.IsZero() {
					field.SetString(fromAddress)
				}
			case fieldName == "SubaccountId": // parsed from context "from"

			// recursively look for internal structs
			case fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && !isComplexValue(fieldT.Elem().String()): // pointer to struct, must be initialized
				if err := fillSenderInStruct(field.Interface()); err != nil {
					return fmt.Errorf("can't fill sender in struct %s: %w", fieldName, err)
				}
			case fieldT.Kind() == reflect.Struct && !isComplexValue(fieldT.String()): // struct
				if err := fillSenderInStruct(field.Addr().Interface()); err != nil {
					return fmt.Errorf("can't fill sender in struct %s: %w", fieldName, err)
				}
			}
		}
		return nil
	}

	return fillSenderInStruct(reflect.ValueOf(msg).Interface())
}

// Parses arguments 1-1 from flags (if corresponding mapping is found in flagsMap) or from args (by position index). Skips already set non-zero fields.
func ParseFieldsFromFlagsAndArgs(
	msg proto.Message, flagsMap FlagsMapping, argsMap ArgsMapping, flags *pflag.FlagSet, args []string, ctx grpc.ClientConn,
) error {
	nextArgIdx := 0
	return parseStruct(reflect.ValueOf(msg).Interface(), flagsMap, argsMap, flags, args, ctx, &nextArgIdx)
}

// parseStruct handles parsing a struct recursively
func parseStruct(
	msg any, flagsMap FlagsMapping, argsMap ArgsMapping, flags *pflag.FlagSet, args []string, ctx grpc.ClientConn, nextArgIdx *int,
) error {
	v := reflect.ValueOf(msg).Elem()
	t := v.Type()
	return parseStructFields(v, t, flagsMap, argsMap, flags, args, ctx, nextArgIdx)
}

func parseStructFields(
	v reflect.Value, t reflect.Type,
	flagsMap FlagsMapping, argsMap ArgsMapping, flags *pflag.FlagSet, args []string, ctx grpc.ClientConn, nextArgIdx *int,
) error {
	for i := 0; i < t.NumField(); i++ {
		if err := parseField(v, t, i, flagsMap, argsMap, flags, args, ctx, nextArgIdx); err != nil {
			return err
		}
	}
	return nil
}

func parseField(
	v reflect.Value, t reflect.Type, i int,
	flagsMap FlagsMapping, argsMap ArgsMapping, flags *pflag.FlagSet, args []string, ctx grpc.ClientConn, nextArgIdx *int,
) error {
	field := v.Field(i)
	fieldT := field.Type()
	fieldName := t.Field(i).Name

	_, hasFlagT := flagsMap[fieldName]
	if hasFlagT {
		return handleBasicField(field, t.Field(i), fieldName, flagsMap, argsMap, flags, args, ctx, nextArgIdx)
	}

	if isAnyType(fieldT) {
		return handleAnyType(field, fieldName, flagsMap, argsMap, flags, args, ctx, nextArgIdx)
	}

	if isPtrToStruct(fieldT) {
		err := parseStruct(field.Interface(), flagsMap, argsMap, flags, args, ctx, nextArgIdx)
		if err != nil {
			return fmt.Errorf("can't parse internal struct %s: %w", fieldName, err)
		}
		return nil
	}

	if isStruct(fieldT) {
		err := parseStruct(field.Addr().Interface(), flagsMap, argsMap, flags, args, ctx, nextArgIdx)
		if err != nil {
			return fmt.Errorf("can't parse internal struct %s: %w", fieldName, err)
		}
		return nil
	}

	return handleBasicField(field, t.Field(i), fieldName, flagsMap, argsMap, flags, args, ctx, nextArgIdx)
}

// isAnyType checks if the field type is an Any type
func isAnyType(fieldT reflect.Type) bool {
	return fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && fieldT.Elem().String() == "types.Any"
}

// isPtrToStruct checks if the field type is a pointer to a struct
func isPtrToStruct(fieldT reflect.Type) bool {
	return fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && !isComplexValue(fieldT.Elem().String())
}

// isStruct checks if the field type is a struct
func isStruct(fieldT reflect.Type) bool {
	return fieldT.Kind() == reflect.Struct && !isComplexValue(fieldT.String())
}

// handleAnyType handles parsing an Any type field
func handleAnyType(
	field reflect.Value,
	fieldName string,
	flagsMap FlagsMapping,
	argsMap ArgsMapping,
	flags *pflag.FlagSet,
	args []string,
	ctx grpc.ClientConn,
	nextArgIdx *int,
) error {
	anyFieldInterface := field.Elem().Interface()
	anyField, ok := anyFieldInterface.(codectypes.Any)
	if !ok {
		return fmt.Errorf("field %s is not of type codectypes.Any", fieldName)
	}
	concreteField := anyField.GetCachedValue()
	if err := parseStruct(concreteField, flagsMap, argsMap, flags, args, ctx, nextArgIdx); err != nil {
		return fmt.Errorf("can't parse internal Any struct %s: %w", fieldName, err)
	}
	parsedAnyField, err := codectypes.NewAnyWithValue(concreteField.(proto.Message))
	if err != nil {
		return fmt.Errorf("can't construct Any struct from parsed %s: %w", fieldName, err)
	}
	field.Set(reflect.ValueOf(parsedAnyField))
	return nil
}

// handleBasicField handles parsing a basic field type
func handleBasicField(
	field reflect.Value,
	fieldType reflect.StructField,
	fieldName string,
	flagsMap FlagsMapping,
	argsMap ArgsMapping,
	flags *pflag.FlagSet,
	args []string,
	ctx grpc.ClientConn,
	nextArgIdx *int,
) error {
	val, found, err := getFieldValue(field, fieldName, flagsMap, argsMap, flags, args, ctx, nextArgIdx)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return parseFieldFromString(field, fieldType, val)
}

func getFieldValue(
	field reflect.Value,
	fieldName string,
	flagsMap FlagsMapping,
	argsMap ArgsMapping,
	flags *pflag.FlagSet,
	args []string,
	ctx grpc.ClientConn,
	nextArgIdx *int,
) (string, bool, error) {
	// Try to get value from flag
	flagTransform, hasFlagT := flagsMap[fieldName]
	if !hasFlagT {
		return getValueFromArgOrPositional(field, fieldName, argsMap, args, ctx, nextArgIdx)
	}

	flagVal, err := getValueFromFlag(flagTransform, flags, ctx, fieldName)
	if err != nil {
		return "", false, err
	}
	if flagVal != "" {
		return flagVal, true, nil
	}

	return "", false, nil
}

// getValueFromArgOrPositional tries to get a value from an argument or positional argument
func getValueFromArgOrPositional(
	field reflect.Value,
	fieldName string,
	argsMap ArgsMapping,
	args []string,
	ctx grpc.ClientConn,
	nextArgIdx *int,
) (string, bool, error) {
	// Try to get value from arg mapping
	val, found, err := tryGetValueFromArg(fieldName, argsMap, args, ctx)
	if err != nil || found {
		return val, found, err
	}

	// Try to get value from positional arg
	if field.IsZero() || isZeroNumber(field) {
		return tryGetValueFromPositionalArg(args, nextArgIdx)
	}

	return "", false, nil
}

// tryGetValueFromArg attempts to get a value from the argument mapping
func tryGetValueFromArg(
	fieldName string,
	argsMap ArgsMapping,
	args []string,
	ctx grpc.ClientConn,
) (string, bool, error) {
	argTransform, hasArgT := argsMap[fieldName]
	if hasArgT && len(args) > argTransform.Index {
		argVal, err := getValueFromArg(argTransform, args, ctx, fieldName)
		if err != nil {
			return "", false, err
		}
		if argVal != "" {
			return argVal, true, nil
		}
	}
	return "", false, nil
}

// tryGetValueFromPositionalArg attempts to get a value from a positional argument
func tryGetValueFromPositionalArg(
	args []string,
	nextArgIdx *int,
) (string, bool, error) {
	posVal, ok := getValueFromPositionalArg(args, nextArgIdx)
	if ok {
		return posVal, true, nil
	}
	return "", false, nil
}

// getValueFromFlag gets a value from a flag
func getValueFromFlag(flagTransform Flag, flags *pflag.FlagSet, ctx grpc.ClientConn, fieldName string) (string, error) {
	if flagTransform.Flag == "" {
		// Special case to leave msg field zero-initialized
		return "", nil
	}
	flag := flags.Lookup(flagTransform.Flag)
	if flag == nil || (!flag.Changed && !flagTransform.UseDefaultIfOmitted) {
		// Flag not found, or not set and we don't want its default value
		return "", nil
	}
	val := flag.Value.String()
	if flagTransform.Transform != nil {
		valT, err := flagTransform.Transform(val, ctx)
		if err != nil {
			return "", fmt.Errorf("error during transforming flag %s: %w", fieldName, err)
		}
		val = fmt.Sprintf("%v", valT)
	}
	return val, nil
}

// getValueFromArg gets a value from an argument
func getValueFromArg(argTransform Arg, args []string, ctx grpc.ClientConn, fieldName string) (string, error) {
	val := args[argTransform.Index]
	if argTransform.Transform != nil {
		valT, err := argTransform.Transform(val, ctx)
		if err != nil {
			return "", fmt.Errorf("error during transforming argument %s: %w", fieldName, err)
		}
		val = fmt.Sprintf("%v", valT)
	}
	args[argTransform.Index] = "" // Mark arg as used by setting it to ""
	return val, nil
}

// getValueFromPositionalArg gets a value from a positional argument
func getValueFromPositionalArg(args []string, nextArgIdx *int) (string, bool) {
	for len(args) > *nextArgIdx {
		val := args[*nextArgIdx]
		*nextArgIdx++
		if val != "" {
			return val, true
		}
	}
	return "", false
}

func parseFieldFromString(fVal reflect.Value, fType reflect.StructField, val string) error {
	switch fVal.Type().Kind() {
	case reflect.Bool:
		b, err := parseBool(val, fType.Name)
		if err != nil {
			return err
		}
		fVal.SetBool(b)
		return nil
	// SetUint allows anyof type u8, u16, u32, u64, and uint
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		u, err := parseUint(val, fType.Name)
		if err != nil {
			return err
		}
		fVal.SetUint(u)
		return nil
	// SetInt allows anyof type i8,i16,i32,i64 and int
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		typeStr := fVal.Type().String()
		var i int64
		var err error
		if typeStr == "time.Duration" {
			dur, err2 := time.ParseDuration(val)
			i, err = int64(dur), err2
		} else {
			i, err = parseInt(val, fType.Name)
		}
		if err != nil {
			return err
		}
		fVal.SetInt(i)
		return nil
	case reflect.String:
		s := parseDenom(val)
		fVal.SetString(s)
		return nil
	case reflect.Slice:
		typeStr := fVal.Type().String()
		if typeStr == "types.Coins" {
			coins, err := parseCoins(val, fType.Name)
			if err != nil {
				return err
			}
			fVal.Set(reflect.ValueOf(coins))
			return nil
		}
		switch fVal.Type().Elem().Name() {
		case "string":
			fVal.Set(reflect.ValueOf(parseStrings(val)))
			return nil
		}
	case reflect.Ptr:
		fVal.Set(reflect.New(fVal.Type().Elem()))
		return parseFieldFromString(fVal.Elem(), fType, val)
	case reflect.Struct:
		typeStr := fVal.Type().String()
		var v any
		var err error
		switch typeStr {
		case "types.Coin":
			v, err = parseCoin(val, fType.Name)
		case "math.Int":
			v, err = parseSdkInt(val, fType.Name)
		case "math.LegacyDec":
			v, err = parseLegacyDec(val, fType.Name)
		case "v2.OpenNotionalCap":
			v, err = parseOpenNotionalCap(val, fType.Name)
		default:
			return fmt.Errorf("struct field type not recognized. Got type %v", fType)
		}
		if err != nil {
			return err
		}
		fVal.Set(reflect.ValueOf(v))
		return nil
	}
	return fmt.Errorf("field type not recognized. Got type %v", fType)
}

func parseBool(arg, fieldName string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "true", "t", "1":
		return true, nil
	case "false", "f", "0":
		return false, nil
	default:
		return false, fmt.Errorf("could not parse %s as bool for field %s", arg, fieldName)
	}
}

func parseUint(arg, fieldName string) (uint64, error) {
	v, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as uint for field %s: %w", arg, fieldName, err)
	}
	return v, nil
}

func parseInt(arg, fieldName string) (int64, error) {
	v, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as int for field %s: %w", arg, fieldName, err)
	}
	return v, nil
}

func parseDenom(arg string) string {
	return strings.TrimSpace(arg)
}

func parseCoin(arg, fieldName string) (sdk.Coin, error) {
	coin, err := sdk.ParseCoinNormalized(arg)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("could not parse %s as sdk.Coin for field %s: %w", arg, fieldName, err)
	}
	return coin, nil
}

func parseCoins(arg, fieldName string) (sdk.Coins, error) {
	coins, err := sdk.ParseCoinsNormalized(arg)
	if err != nil {
		return sdk.Coins{}, fmt.Errorf("could not parse %s as sdk.Coins for field %s: %w", arg, fieldName, err)
	}
	return coins, nil
}

func parseStrings(arg string) []string {
	return strings.Split(arg, ",")
}

func parseSdkInt(arg, fieldName string) (math.Int, error) {
	i, ok := math.NewIntFromString(arg)
	if !ok {
		return math.Int{}, fmt.Errorf("could not parse %s as math.Int for field %s", arg, fieldName)
	}
	return i, nil
}

func parseLegacyDec(arg, fieldName string) (math.LegacyDec, error) {
	i, err := math.LegacyNewDecFromStr(arg)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("could not parse %s as math.LegacyDec for field %s: %w", arg, fieldName, err)
	}
	return i, nil
}

func parseOpenNotionalCap(arg, fieldName string) (exchangev2.OpenNotionalCap, error) {
	openNotionalCap, err := GetOpenNotionalCapFromString(arg)
	if err != nil {
		return exchangev2.OpenNotionalCap{}, fmt.Errorf("could not parse %s as OpenNotionalCap for field %s: %w", arg, fieldName, err)
	}
	return openNotionalCap, nil
}

func isFilledFromCtx(fName string) bool {
	switch fName {
	case "Sender", "SubaccountId", "FeeRecipient", "Proposer", "Granter", "SourceSubaccountId":
		return true
	}
	return false
}

func isComplexValue(typeName string) bool {
	switch typeName {
	case "types.Coin",
		"types.Any",
		"math.LegacyDec",
		"math.Int",
		"math.Dec":
		return true
	}
	return false
}

// isZeroNumber determines if math.LegacyDec or math.Int has zero value
func isZeroNumber(field reflect.Value) bool {
	if field.Type().String() == "math.Dec" ||
		field.Type().String() == "math.Int" ||
		field.Type().String() == "math.LegacyDec" {
		isNil := field.MethodByName("IsNil").Call([]reflect.Value{})[0].Bool()
		if isNil {
			return true
		}
		isZero := field.MethodByName("IsZero").Call([]reflect.Value{})[0].Bool()
		if isZero {
			return true
		}
	}
	return false
}

// types.QueryXXXRequest{} -> queryClient.XXX()
func parseExpectedQueryFnName(message proto.Message) string {
	s := reflect.TypeOf(message).Elem().String()
	// handle some non-std queries
	var prefixTrimmed string
	if strings.Contains(s, "Query") {
		prefixTrimmed = strings.Split(s, "Query")[1]
	} else {
		prefixTrimmed = strings.Split(s, ".")[1]
	}
	suffixTrimmed := strings.TrimSuffix(prefixTrimmed, "Request")
	return suffixTrimmed
}

func GetOpenNotionalCapFromString(orig string) (exchangev2.OpenNotionalCap, error) {
	if orig == "" {
		return exchangev2.OpenNotionalCap{}, nil
	}
	if orig == "uncapped" {
		return exchangev2.OpenNotionalCap{
			Cap: &exchangev2.OpenNotionalCap_Uncapped{
				Uncapped: &exchangev2.OpenNotionalCapUncapped{},
			},
		}, nil
	}
	openNotionalCapValue, err := math.LegacyNewDecFromStr(orig)
	if err != nil {
		return exchangev2.OpenNotionalCap{}, err
	}
	return exchangev2.OpenNotionalCap{
		Cap: &exchangev2.OpenNotionalCap_Capped{
			Capped: &exchangev2.OpenNotionalCapCapped{
				Value: openNotionalCapValue,
			},
		},
	}, nil
}
