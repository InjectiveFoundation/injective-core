package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
)

// Generic function to serialize a slice of protobuf messages
func ConvertToString[T proto.Message](msgs []T) string {
	var sb strings.Builder
	for _, msg := range msgs {
		// Use reflection to get the fields and their values
		v := reflect.ValueOf(msg).Elem()
		typeOfMsg := v.Type()

		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldName := typeOfMsg.Field(i).Name
			fieldValue := field.Interface()

			sb.WriteString(fmt.Sprintf("%s:%v ", fieldName, fieldValue))
		}
		sb.WriteString("\n") // Add a newline separator between messages
	}
	return sb.String()
}
