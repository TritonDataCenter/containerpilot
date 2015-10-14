package main

import (
	"flag"
	"os"
	"testing"
)

func TestArgParse(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"this", "-poll", "20", "/root/examples/test.sh", "doStuff", "--debug"}
	config := parseArgs()
	if config.pollTime != 20 {
		t.Errorf("Expected pollTime to be 20 but got: %d", config.pollTime)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/root/examples/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}
}
