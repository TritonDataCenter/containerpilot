// Package decode contains helper functions for turning  mapstructure
// interfaces into simpler structs for configs
package decode

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// ToStruct decodes a raw interface{} into the target struct
func ToStruct(raw interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           result,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(raw)
}

// ToSlice converts an interface{} to a slice of interfaces{}
func ToSlice(raw interface{}) []interface{} {
	if raw == nil {
		return nil
	}
	var arr []interface{}
	switch reflect.TypeOf(raw).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(raw)
		for i := 0; i < s.Len(); i++ {
			v := s.Index(i)
			if !v.IsNil() {
				arr = append(arr, v.Interface())
			}
		}
		return arr
	}
	return nil
}

// ToStrings converts the given interface{} to a []string, or returns an error.
// In the case where the argument is a string already, will wrap the arg in
// a slice.
func ToStrings(raw interface{}) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	switch t := raw.(type) {
	case string:
		return []string{t}, nil
	case []string:
		return t, nil
	case []interface{}:
		return interfaceToStringArray(t), nil
	default:
		return nil, fmt.Errorf("unexpected argument type: %T", t)
	}
}

func interfaceToString(raw interface{}) string {
	switch t := raw.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func interfaceToStringArray(rawArray []interface{}) []string {
	if len(rawArray) == 0 {
		return nil
	}
	var stringArray []string
	for _, raw := range rawArray {
		stringArray = append(stringArray, interfaceToString(raw))
	}
	return stringArray
}
