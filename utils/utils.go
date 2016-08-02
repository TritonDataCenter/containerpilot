package utils

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

// DecodeRaw decodes a raw interface into the target structure
func DecodeRaw(raw interface{}, result interface{}) error {
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

// ToStringArray converts the given interface to a []string if possible
func ToStringArray(raw interface{}) ([]string, error) {
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
		return nil, fmt.Errorf("Unexpected argument type: %T", t)
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
	if rawArray == nil || len(rawArray) == 0 {
		return nil
	}
	var stringArray []string
	for _, raw := range rawArray {
		stringArray = append(stringArray, interfaceToString(raw))
	}
	return stringArray
}
