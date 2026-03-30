package evaluator

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestMatchCapturesArrayValues(t *testing.T) {
	env, result := evalSource(t, `
pair = [3, 3]
result = match pair {
    [x, x] -> "same:" + str(x)
    _ -> "different"
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
	if !ok || strValue.Value != "same:3" {
		t.Fatalf("expected same:3, got %s", value.Inspect())
	}
}

func TestMatchHashPatternWithGuard(t *testing.T) {
	env, result := evalSource(t, `
payload = {"kind": "ok", "value": 7}
result = match payload {
    {"kind": "ok", "value": value} if value > 5 -> "high:" + str(value)
    {"kind": "ok", "value": value} -> "low:" + str(value)
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
	if !ok || strValue.Value != "high:7" {
		t.Fatalf("expected high:7, got %s", value.Inspect())
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

func TestGoStatementDrainsGoroutinesAfterWait(t *testing.T) {
	baseline := runtime.NumGoroutine()

	_, result := evalSource(t, `
fn noop(v) {
    return v
}

for i in range(50) {
    go noop(i)
}
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	time.Sleep(20 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > baseline+2 {
		t.Fatalf("possible goroutine leak: baseline=%d after=%d", baseline, after)
	}
}

func TestFSLibraryReadWriteAndStat(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	filePath := dir + "/sample.txt"

	env, result := evalSource(t, fmt.Sprintf(`
fs.write_file("%s", "hello")
content = fs.read_file("%s")
exists = fs.exists("%s")
info = fs.stat("%s")
`, filePath, filePath, filePath, filePath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	content, _ := env.Get("content")
	if content.Inspect() != "hello" {
		t.Fatalf("expected content=hello, got %s", content.Inspect())
	}

	exists, _ := env.Get("exists")
	if exists.Inspect() != "true" {
		t.Fatalf("expected exists=true, got %s", exists.Inspect())
	}

	info, _ := env.Get("info")
	infoHash, ok := info.(*object.Hash)
	if !ok {
		t.Fatalf("expected hash info, got %T", info)
	}
	size := infoHash.Pairs["size"].Value
	if size.Inspect() != "5" {
		t.Fatalf("expected size=5, got %s", size.Inspect())
	}
}

func TestHTTPLibraryClientAndServer(t *testing.T) {
	env, result := evalSource(t, `
server = http.server.new()
server.route("/hello", "world")
addr = server.start("127.0.0.1:0")
resp = http.client.get("http://" + addr + "/hello")
server.stop()
body = resp.body
status = resp.status_code
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	body, _ := env.Get("body")
	if body.Inspect() != "world" {
		t.Fatalf("expected body=world, got %s", body.Inspect())
	}

	status, _ := env.Get("status")
	if status.Inspect() != "200" {
		t.Fatalf("expected status=200, got %s", status.Inspect())
	}
}
