package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/pflag"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
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
			concreteStruct := field.Elem().Interface().(codectypes.Any)
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
func parseFieldsFromFlagsAndArgs(msg proto.Message, flagsMap FlagsMapping, argsMap ArgsMapping, flags *pflag.FlagSet, args []string, ctx grpc.ClientConn) error {

	nextArgIdx := 0

	var parseStruct func(any) error
	parseStruct = func(msg any) error {
		v := reflect.ValueOf(msg).Elem()
		t := v.Type()
		for i := 0; i < v.Type().NumField(); i++ {
			field := v.Field(i)
			fieldT := field.Type()
			switch {
			case fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && fieldT.Elem().String() == "types.Any": // proto-encoded type, never nil
				anyField := field.Elem().Interface().(codectypes.Any)
				concreteField := anyField.GetCachedValue()
				if err := parseStruct(concreteField); err != nil {
					return fmt.Errorf("can't parse internal Any struct %s: %w", t.Field(i).Name, err)
				}
				if parsedAnyField, err := codectypes.NewAnyWithValue(concreteField.(proto.Message)); err != nil {
					return fmt.Errorf("can't construct Any struct from parsed %s: %w", t.Field(i).Name, err)
				} else {
					field.Set(reflect.ValueOf(parsedAnyField))
				}
			// recursively look for internal structs
			case fieldT.Kind() == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct && !isComplexValue(fieldT.Elem().String()): // pointer to struct, must be initialized
				if err := parseStruct(field.Interface()); err != nil {
					return fmt.Errorf("can't parse internal struct %s: %w", t.Field(i).Name, err)
				}
			case fieldT.Kind() == reflect.Struct && !isComplexValue(fieldT.String()): // struct
				if err := parseStruct(field.Addr().Interface()); err != nil {
					return fmt.Errorf("can't parse internal struct %s: %w", t.Field(i).Name, err)
				}
			default:
				val := ""
				flagTransform, hasFlagT := flagsMap[t.Field(i).Name]
				argTransform, hasArgT := argsMap[t.Field(i).Name]

				switch {
				case hasFlagT: // flags mapping
					if flagTransform.Flag == "" { // special case to leave msg field zero-initialized
						continue
					}
					flag := flags.Lookup(flagTransform.Flag)
					if flag == nil || (!flag.Changed && !flagTransform.UseDefaultIfOmitted) { // flag not found, or not set and we don't want it's default value
						continue
					}
					val = flag.Value.String()
					if flagTransform.Transform != nil {
						if valT, err := flagTransform.Transform(val, ctx); err != nil {
							return fmt.Errorf("error during transforming flag %s: %w", t.Field(i).Name, err)
						} else {
							val = fmt.Sprintf("%v", valT)
						}
					}
				case hasArgT && len(args) > argTransform.Index: // args mapping
					val = args[argTransform.Index]
					if argTransform.Transform != nil {
						if valT, err := argTransform.Transform(val, ctx); err != nil {
							return fmt.Errorf("error during transforming argument %s: %w", t.Field(i).Name, err)
						} else {
							val = fmt.Sprintf("%v", valT)
						}
					}
					args[argTransform.Index] = "" // mark arg as used by setting it to ""
				case field.IsZero(), isZeroNumber(field): // fill only zero fields from args
					for len(args) > nextArgIdx { // skip used arguments
						val = args[nextArgIdx]
						nextArgIdx++
						if val != "" {
							break
						}
					}
				default:
					continue
				}

				if err := parseFieldFromString(field, t.Field(i), val); err != nil {
					return err
				}
			}

		}
		return nil
	}
	return parseStruct(reflect.ValueOf(msg).Interface())
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

func parseSdkInt(arg, fieldName string) (sdkmath.Int, error) {
	i, ok := sdk.NewIntFromString(arg)
	if !ok {
		return sdkmath.Int{}, fmt.Errorf("could not parse %s as math.Int for field %s", arg, fieldName)
	}
	return i, nil
}

func parseLegacyDec(arg, fieldName string) (sdk.Dec, error) {
	i, err := sdk.NewDecFromStr(arg)
	if err != nil {
		return sdk.Dec{}, fmt.Errorf("could not parse %s as math.LegacyDec for field %s: %w", arg, fieldName, err)
	}
	return i, nil
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

// isZeroNumber determines if sdk.Dec or sdkmath.Int has zero value
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
