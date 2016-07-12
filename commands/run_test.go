package commands

import (
	"testing"
)

func TestRunSuccess(t *testing.T) {
	cmd1 := StrToCmd("./testdata/test.sh doStuff --debug")
	if exitCode, _ := Run(cmd1); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
	cmd2 := ArgsToCmd([]string{"./testdata/test.sh", "doStuff", "--debug"})
	if exitCode, _ := Run(cmd2); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
}

func BenchmarkRunSuccess(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Run(StrToCmd("./testdata/test.sh doNothing"))
	}
}

func TestRunFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic but did not.")
		}
	}()
	cmd := StrToCmd("./testdata/test.sh failStuff --debug")
	if exitCode, _ := Run(cmd); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
	}
}

func TestRunNothing(t *testing.T) {
	if code, err := Run(StrToCmd("")); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
}

func TestReuseCmd(t *testing.T) {
	cmd := ArgsToCmd([]string{"true"})
	if code, err := Run(cmd); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
	cmd = ArgsToCmd(cmd.Args)
	if code, err := Run(cmd); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
}
