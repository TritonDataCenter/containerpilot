package main

import (
	"log"
	"os"
)

// TestCommand is a test that can be run by the probe
type TestCommand func([]string) bool

// AllTests maps a test name to its TestCommand function
var AllTests map[string]TestCommand

func runTest(testName string, args []string) {
	// Register Tests
	AllTests = map[string]TestCommand{
		"test_consul":           TestConsul,
		"test_discovery":        TestDiscovery,
		"test_sighup_deadlock":  TestSighupDeadlock,
		"test_sighup_prestart":  TestSigHupPrestart,
		"test_sigterm":          TestSigterm,
		"test_sigusr1_prestart": TestSigUsr1Prestart,
	}

	if test := AllTests[testName]; test != nil {
		args := os.Args[2:]
		if success := test(args); success {
			os.Exit(0)
		}
	}
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Incorrect number of arguments. Expected TEST_NAME [ ARGUMENTS ... ]")
	}
	runTest(os.Args[1], os.Args[2:])
}
