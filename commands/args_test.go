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
	exec, args, err = ParseArgs([]string{"/testdata/test.sh", "arg1"})
	validateParsing(t, exec, "/testdata/test.sh", args, []string{"arg1"}, err, nil)

	// interface args ok
	exec, args, err = ParseArgs([]interface{}{"/testdata/test.sh", "arg1"})
	validateParsing(t, exec, "/testdata/test.sh", args, []string{"arg1"}, err, nil)

	// map of bools args return error
	exec, args, err = ParseArgs([]bool{true})
	validateParsing(t, exec, "", args, nil,
		err, errors.New("received zero-length argument"))
}

func validateParsing(t *testing.T, exec, expectedExec string,
	args, expectedArgs []string, err, expectedErr error) {
	if !reflect.DeepEqual(err, expectedErr) { //}err != expectedErr {
		t.Errorf("expected %s but got %s", expectedErr, err)
		return
	}
	if exec != expectedExec {
		t.Errorf("executable not parsed: %s != %s", exec, expectedExec)
		return
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("args not parsed: %s != %s", args, expectedArgs)
		return
	}
}
