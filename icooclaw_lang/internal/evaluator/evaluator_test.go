package evaluator

import (
	"testing"

	"github.com/issueye/icooclaw_lang/internal/lexer"
	"github.com/issueye/icooclaw_lang/internal/object"
	"github.com/issueye/icooclaw_lang/internal/parser"
)

func evalSource(t *testing.T, input string) (*object.Environment, object.Object) {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}

	env := object.NewEnvironment()
	result := Eval(program, env)
	env.Wait()
	return env, result
}

func TestInlineBlockStatementsDoNotHang(t *testing.T) {
	env, result := evalSource(t, `
x = 0
if true { x = 1 }
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	value, ok := env.Get("x")
	if !ok {
		t.Fatal("x not found in environment")
	}

	intValue, ok := value.(*object.Integer)
	if !ok || intValue.Value != 1 {
		t.Fatalf("expected x=1, got %s", value.Inspect())
	}
}

func TestMatchExpressionAssignsValue(t *testing.T) {
	env, result := evalSource(t, `
x = 2
result = match x {
    1 -> "one"
    2 -> "two"
    _ -> "other"
}
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	value, ok := env.Get("result")
	if !ok {
		t.Fatal("result not found in environment")
	}

	strValue, ok := value.(*object.String)
	if !ok || strValue.Value != "two" {
		t.Fatalf("expected result=two, got %s", value.Inspect())
	}
}

func TestConstReassignmentReturnsError(t *testing.T) {
	_, result := evalSource(t, `
const A = 1
A = 2
`)

	if !object.IsError(result) {
		t.Fatalf("expected constant reassignment error, got %#v", result)
	}
}

func TestGoStatementRunsFunctionCall(t *testing.T) {
	env, result := evalSource(t, `
x = 0
fn set() {
    x = 1
}
go set()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	value, ok := env.Get("x")
	if !ok {
		t.Fatal("x not found in environment")
	}

	intValue, ok := value.(*object.Integer)
	if !ok || intValue.Value != 1 {
		t.Fatalf("expected x=1 after goroutine, got %s", value.Inspect())
	}
}

func TestTypeBuiltinRemainsCallable(t *testing.T) {
	env, result := evalSource(t, `
kind = type(42)
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	value, ok := env.Get("kind")
	if !ok {
		t.Fatal("kind not found in environment")
	}

	strValue, ok := value.(*object.String)
	if !ok || strValue.Value != "INTEGER" {
		t.Fatalf("expected kind=INTEGER, got %s", value.Inspect())
	}
}
