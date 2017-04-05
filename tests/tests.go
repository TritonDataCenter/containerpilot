package tests

import (
	"github.com/flynn/json5"
)

// DecodeRawToSlice supports testing NewConfig functions, which never receive the
// raw string, but instead get the []interface{} parsed in the config package.
// This mimics the behavior of the config package, but bails if we've written
// bad test JSON
func DecodeRawToSlice(input string) []interface{} {
	testCfg := []byte(input)
	var raw []interface{}
	if err := json5.Unmarshal(testCfg, &raw); err != nil {
		// this is an error in our test, not in the tested code
		panic("unexpected error decoding test fixture JSON5:\n" + err.Error())
	}
	return raw
}

// DecodeRaw supports testing NewConfig functions, which never receive the
// raw string, but instead get the interface{} parsed in the config package.
// This mimics the behavior of the config package, but bails if we've written
// bad test JSON
func DecodeRaw(input string) interface{} {
	testCfg := []byte(input)
	var raw interface{}
	if err := json5.Unmarshal(testCfg, &raw); err != nil {
		// this is an error in our test, not in the tested code
		panic("unexpected error decoding test fixture JSON5:\n" + err.Error())
	}
	return raw
}
