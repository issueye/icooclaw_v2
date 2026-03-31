package builtin

import (
	"os"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestInputUsesSharedReaderAcrossCalls(t *testing.T) {
	originalStdin := os.Stdin
	t.Cleanup(func() {
		os.Stdin = originalStdin
		stdinReaderMu.Lock()
		stdinReader = nil
		stdinSource = nil
		stdinReaderMu.Unlock()
	})

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer readPipe.Close()

	if _, err := writePipe.WriteString("first\nsecond\n"); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	writePipe.Close()

	os.Stdin = readPipe
	stdinReaderMu.Lock()
	stdinReader = nil
	stdinSource = nil
	stdinReaderMu.Unlock()

	inputBuiltin, ok := coreBuiltins()["input"].(*object.Builtin)
	if !ok {
		t.Fatalf("input builtin not found")
	}

	first := inputBuiltin.Fn(object.NewEnvironment())
	firstValue, ok := first.(*object.String)
	if !ok {
		t.Fatalf("first result type = %T", first)
	}
	if firstValue.Value != "first" {
		t.Fatalf("first value = %q", firstValue.Value)
	}

	second := inputBuiltin.Fn(object.NewEnvironment())
	secondValue, ok := second.(*object.String)
	if !ok {
		t.Fatalf("second result type = %T", second)
	}
	if secondValue.Value != "second" {
		t.Fatalf("second value = %q", secondValue.Value)
	}
}
