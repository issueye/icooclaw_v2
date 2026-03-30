package parser

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
)

func TestTrailingCommaFormsDoNotHang(t *testing.T) {
	source := `
fn build_payload(
    name,
    count,
) {
    data = {
        "name": name,
        "count": count,
        "items": [
            "a",
            "b",
        ],
    }

    return data
}

payload = build_payload(
    "icooclaw",
    2,
)
`

	program, errs := parseProgramWithTimeout(t, source, 2*time.Second)
	if len(errs) != 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if program == nil || len(program.Statements) == 0 {
		t.Fatal("expected parsed statements")
	}
}

func TestLargeHashLiteralWithTrailingCommasParsesQuickly(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("payload = {\n")
	for i := 0; i < 400; i++ {
		builder.WriteString(fmt.Sprintf("    \"key_%d\": [\n        %d,\n        %d,\n    ],\n", i, i, i+1))
	}
	builder.WriteString("}\n")

	program, errs := parseProgramWithTimeout(t, builder.String(), 2*time.Second)
	if len(errs) != 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	if program == nil || len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
}

func TestParseProgramRepeatedlyStaysStable(t *testing.T) {
	source := `
fn call_all(
    a,
    b,
) {
    return {
        "pair": [
            a,
            b,
        ],
        "meta": {
            "sum": a + b,
        },
    }
}

result = call_all(
    1,
    2,
)
`

	for i := 0; i < 100; i++ {
		program, errs := parseProgramWithTimeout(t, source, 500*time.Millisecond)
		if len(errs) != 0 {
			t.Fatalf("iteration %d unexpected parser errors: %v", i, errs)
		}
		if program == nil || len(program.Statements) == 0 {
			t.Fatalf("iteration %d expected parsed statements", i)
		}
	}
}

func parseProgramWithTimeout(t *testing.T, source string, timeout time.Duration) (*ast.Program, []string) {
	t.Helper()

	type result struct {
		program *ast.Program
		errors  []string
	}

	done := make(chan result, 1)
	go func() {
		l := lexer.New(source)
		p := New(l)
		done <- result{
			program: p.ParseProgram(),
			errors:  p.Errors(),
		}
	}()

	select {
	case parsed := <-done:
		return parsed.program, parsed.errors
	case <-time.After(timeout):
		t.Fatalf("parser timed out after %s", timeout)
		return nil, nil
	}
}
