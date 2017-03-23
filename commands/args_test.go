package commands

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {

	// nil args should return error
	exec, args, err := ParseArgs(nil)
	validateParsing(t, exec, "", args, nil,
		err, errors.New("received zero-length argument"))

	// string args ok
	exec, args, err = ParseArgs("/testdata/test.sh arg1")
	validateParsing(t, exec, "/testdata/test.sh", args, []string{"arg1"}, err, nil)

	// array args ok
	exec, args, err = ParseArgs([]string{"/testdata/test.sh", "arg2"})
	validateParsing(t, exec, "/testdata/test.sh", args, []string{"arg2"}, err, nil)

	// interface args ok
	exec, args, err = ParseArgs([]interface{}{"/testdata/test.sh", "arg3"})
	validateParsing(t, exec, "/testdata/test.sh", args, []string{"arg3"}, err, nil)

	// map of bools args return error
	exec, args, err = ParseArgs([]bool{true})
	validateParsing(t, exec, "", args, nil,
		err, errors.New("received zero-length argument"))
}

func validateParsing(t *testing.T, exec, expectedExec string,
	args, expectedArgs []string, err, expectedErr error) {
	if !reflect.DeepEqual(err, expectedErr) { //}err != expectedErr {
		t.Fatalf("expected %s but got %s", expectedErr, err)
	}
	if exec != expectedExec {
		t.Fatalf("executable not parsed: %s != %s", exec, expectedExec)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("args not parsed: %s != %s", args, expectedArgs)
	}
}
