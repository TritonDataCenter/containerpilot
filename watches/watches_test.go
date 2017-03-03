package watches

import (
	"context"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

func TestOnChangeCmd(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "1s")
	watch := &Watch{
		exec: cmd1,
	}

	if err := watch.OnChange(context.Background()); err != nil {
		t.Fatalf("Unexpected error OnChange: %s", err)
	}
	// Ensure we can run it more than once
	if err := watch.OnChange(context.Background()); err != nil {
		t.Fatalf("Unexpected error OnChange (x2): %s", err)
	}
}
