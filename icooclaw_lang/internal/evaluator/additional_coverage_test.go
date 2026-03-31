package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
	"github.com/issueye/icooclaw_lang/internal/object"
	"github.com/issueye/icooclaw_lang/internal/parser"
)

func TestCollectionsStringMethodsAndConversions(t *testing.T) {
	env, result := evalSource(t, `
items = [1, 2]
items2 = push(items, 3)
last = pop(items2)
joined = ["a", "b", "c"].join("-")
contains_b = ["a", "b", "c"].contains("b")
parts = "  alpha,beta  ".trim().split(",")
starts = "icooclaw".starts_with("icoo")
ends = "icooclaw".ends_with("law")
hash_len = len({"x": 1, "y": 2})
numbers = range(2, 5)
int_value = int("12")
float_value = float("3.5")
abs_int = abs(-7)
abs_float = abs(-3.5)
hash_keys = keys({"name": "icooclaw", "lang": "is"})
hash_values = values({"name": "icooclaw", "lang": "is"})
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "joined"); got != "a-b-c" {
		t.Fatalf("expected joined=a-b-c, got %s", got)
	}
	if got := testStringValue(t, env, "last"); got != "3" {
		t.Fatalf("expected last=3, got %s", got)
	}
	if got := testStringValue(t, env, "contains_b"); got != "true" {
		t.Fatalf("expected contains_b=true, got %s", got)
	}
	if got := testStringValue(t, env, "starts"); got != "true" {
		t.Fatalf("expected starts=true, got %s", got)
	}
	if got := testStringValue(t, env, "ends"); got != "true" {
		t.Fatalf("expected ends=true, got %s", got)
	}
	if got := testStringValue(t, env, "hash_len"); got != "2" {
		t.Fatalf("expected hash_len=2, got %s", got)
	}
	if got := testStringValue(t, env, "int_value"); got != "12" {
		t.Fatalf("expected int_value=12, got %s", got)
	}
	if got := testStringValue(t, env, "float_value"); got != "3.5" {
		t.Fatalf("expected float_value=3.5, got %s", got)
	}
	if got := testStringValue(t, env, "abs_int"); got != "7" {
		t.Fatalf("expected abs_int=7, got %s", got)
	}
	if got := testStringValue(t, env, "abs_float"); got != "3.5" {
		t.Fatalf("expected abs_float=3.5, got %s", got)
	}

	parts := testArrayValue(t, env, "parts")
	if len(parts.Elements) != 2 || parts.Elements[0].Inspect() != "alpha" || parts.Elements[1].Inspect() != "beta" {
		t.Fatalf("unexpected parts array: %s", parts.Inspect())
	}

	numbers := testArrayValue(t, env, "numbers")
	if len(numbers.Elements) != 3 || numbers.Elements[0].Inspect() != "2" || numbers.Elements[2].Inspect() != "4" {
		t.Fatalf("unexpected numbers array: %s", numbers.Inspect())
	}

	hashKeys := testArrayValue(t, env, "hash_keys")
	if len(hashKeys.Elements) != 2 || !testArrayContains(hashKeys, "name") || !testArrayContains(hashKeys, "lang") {
		t.Fatalf("unexpected hash_keys: %s", hashKeys.Inspect())
	}

	hashValues := testArrayValue(t, env, "hash_values")
	if len(hashValues.Elements) != 2 || !testArrayContains(hashValues, "icooclaw") || !testArrayContains(hashValues, "is") {
		t.Fatalf("unexpected hash_values: %s", hashValues.Inspect())
	}
}

func TestTryCatchAndTrailingCommaHashLiteral(t *testing.T) {
	env, result := evalSource(t, `
fn build_payload() {
    return {
        "name": "icooclaw",
        "features": [
            "match",
            "http",
        ],
    }
}

catch_result = ""

try {
    broken = json.parse("{bad json}")
} catch err {
    catch_result = "caught:" + err
}

payload = build_payload()
payload_name = payload.name
feature_count = payload.features.len()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	catchResult := testStringValue(t, env, "catch_result")
	if !strings.Contains(catchResult, "caught:could not parse json") {
		t.Fatalf("unexpected catch_result: %s", catchResult)
	}
	if got := testStringValue(t, env, "payload_name"); got != "icooclaw" {
		t.Fatalf("expected payload_name=icooclaw, got %s", got)
	}
	if got := testStringValue(t, env, "feature_count"); got != "2" {
		t.Fatalf("expected feature_count=2, got %s", got)
	}
}

func TestFSLibraryDirectoryOperations(t *testing.T) {
	rootDir := filepath.ToSlash(filepath.Join(t.TempDir(), "fs_suite"))
	filePath := rootDir + "/demo.txt"

	env, result := evalSource(t, fmt.Sprintf(`
root = "%s"
file = "%s"

fs.mkdir(root)
fs.write_file(file, "hello")
fs.append_file(file, " world")
content = fs.read_file(file)
entries = fs.read_dir(root)
entry_name = entries[0].name
entry_is_dir = entries[0].is_dir
absolute_file = fs.abs(file)
fs.remove(file)
file_exists_after_remove = fs.exists(file)
fs.remove(root)
root_exists_after_remove = fs.exists(root)
`, rootDir, filePath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "content"); got != "hello world" {
		t.Fatalf("expected content=hello world, got %s", got)
	}
	if got := testStringValue(t, env, "entry_name"); got != "demo.txt" {
		t.Fatalf("expected entry_name=demo.txt, got %s", got)
	}
	if got := testStringValue(t, env, "entry_is_dir"); got != "false" {
		t.Fatalf("expected entry_is_dir=false, got %s", got)
	}
	if got := testStringValue(t, env, "absolute_file"); !filepath.IsAbs(got) {
		t.Fatalf("expected absolute_file to be absolute, got %s", got)
	}
	if got := testStringValue(t, env, "file_exists_after_remove"); got != "false" {
		t.Fatalf("expected file_exists_after_remove=false, got %s", got)
	}
	if got := testStringValue(t, env, "root_exists_after_remove"); got != "false" {
		t.Fatalf("expected root_exists_after_remove=false, got %s", got)
	}

	entries := testArrayValue(t, env, "entries")
	if len(entries.Elements) != 1 {
		t.Fatalf("expected 1 directory entry, got %d", len(entries.Elements))
	}
}

func TestHTTPServerRouteJSONStateAndRequestHeaders(t *testing.T) {
	env, result := evalSource(t, `
fn inspect(req) {
    return {
        "status_code": 204,
        "headers": {
            "X-Echo": req.headers["X-Token"][0],
        },
    }
}

fn empty(req) {
    return null
}

server = http.server.new()
before_running = server.is_running()
server.route_json("/json", {"ok": true, "count": 2})
server.handle("POST", "/inspect", inspect)
server.handle("/empty", empty)
addr = server.start("127.0.0.1:0")
running = server.is_running()
addr_copy = server.addr()
json_resp = http.client.get(server.url("/json"), {"X-Test": "yes"})
decoded = json.parse(json_resp.body)
inspect_resp = http.client.request("POST", server.url("/inspect"), "", {"X-Token": "abc"})
empty_resp = http.client.get(server.url("/empty"))
stats_before_stop = server.stats()
server.stop()
after_running = server.is_running()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "before_running"); got != "false" {
		t.Fatalf("expected before_running=false, got %s", got)
	}
	if got := testStringValue(t, env, "running"); got != "true" {
		t.Fatalf("expected running=true, got %s", got)
	}
	if got := testStringValue(t, env, "after_running"); got != "false" {
		t.Fatalf("expected after_running=false, got %s", got)
	}

	addr := testStringValue(t, env, "addr")
	addrCopy := testStringValue(t, env, "addr_copy")
	if addr == "" || addr != addrCopy {
		t.Fatalf("expected addr and addr_copy to match, got addr=%s addr_copy=%s", addr, addrCopy)
	}

	decoded := testHashValue(t, env, "decoded")
	if decoded.Pairs["ok"].Value.Inspect() != "true" || decoded.Pairs["count"].Value.Inspect() != "2" {
		t.Fatalf("unexpected decoded response: %s", decoded.Inspect())
	}

	inspectResp := testHashValue(t, env, "inspect_resp")
	if inspectResp.Pairs["status_code"].Value.Inspect() != "204" {
		t.Fatalf("expected inspect status=204, got %s", inspectResp.Pairs["status_code"].Value.Inspect())
	}
	inspectHeaders := inspectResp.Pairs["headers"].Value.(*object.Hash)
	echoValues := inspectHeaders.Pairs["X-Echo"].Value.(*object.Array)
	if len(echoValues.Elements) != 1 || echoValues.Elements[0].Inspect() != "abc" {
		t.Fatalf("unexpected X-Echo header: %s", echoValues.Inspect())
	}

	emptyResp := testHashValue(t, env, "empty_resp")
	if emptyResp.Pairs["status_code"].Value.Inspect() != "200" || emptyResp.Pairs["body"].Value.Inspect() != "" {
		t.Fatalf("unexpected empty response: %s", emptyResp.Inspect())
	}

	stats := testHashValue(t, env, "stats_before_stop")
	if stats.Pairs["route_count"].Value.Inspect() != "3" {
		t.Fatalf("expected route_count=3, got %s", stats.Pairs["route_count"].Value.Inspect())
	}
	if stats.Pairs["request_count"].Value.Inspect() != "3" {
		t.Fatalf("expected request_count=3, got %s", stats.Pairs["request_count"].Value.Inspect())
	}
}

func TestOSArgsAndCryptoDecodeError(t *testing.T) {
	env := object.NewEnvironment()
	env.SetCLIContext("examples/demo.is", []string{"input.txt", "--mode=prod", "--verbose", "-p", "8080"})
	result := Eval(parseProgramForTest(t, `
hostname = os.hostname()
temp_dir = os.temp_dir()
argv = os.args()
first_arg = os.arg(0)
missing_arg = os.arg(9)
mode = os.flag("mode")
port = os.flag("p")
verbose = os.has_flag("verbose")
dry_run = os.has_flag("dry-run")
fallback = os.flag_or("config", "default.toml")
script_path = os.script_path()
`), env)
	env.Wait()

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "hostname"); got == "" {
		t.Fatal("expected hostname to be non-empty")
	}
	if got := testStringValue(t, env, "temp_dir"); got == "" {
		t.Fatal("expected temp_dir to be non-empty")
	}
	argv := testArrayValue(t, env, "argv")
	if len(argv.Elements) != 5 {
		t.Fatalf("expected os.args() to contain 5 script arguments, got %d", len(argv.Elements))
	}
	if got := testStringValue(t, env, "first_arg"); got != "input.txt" {
		t.Fatalf("expected first_arg=input.txt, got %s", got)
	}
	if got := testStringValue(t, env, "missing_arg"); got != "null" {
		t.Fatalf("expected missing_arg=null, got %s", got)
	}
	if got := testStringValue(t, env, "mode"); got != "prod" {
		t.Fatalf("expected mode=prod, got %s", got)
	}
	if got := testStringValue(t, env, "port"); got != "8080" {
		t.Fatalf("expected port=8080, got %s", got)
	}
	if got := testStringValue(t, env, "verbose"); got != "true" {
		t.Fatalf("expected verbose=true, got %s", got)
	}
	if got := testStringValue(t, env, "dry_run"); got != "false" {
		t.Fatalf("expected dry_run=false, got %s", got)
	}
	if got := testStringValue(t, env, "fallback"); got != "default.toml" {
		t.Fatalf("expected fallback=default.toml, got %s", got)
	}
	if got := testStringValue(t, env, "script_path"); got != "examples/demo.is" {
		t.Fatalf("expected script_path=examples/demo.is, got %s", got)
	}

	_, cryptoErr := evalSource(t, `
crypto.base_64_decode("%%%")
`)
	if !object.IsError(cryptoErr) {
		t.Fatalf("expected invalid base64 to return error, got %#v", cryptoErr)
	}
}

func TestModuleImportNamespaceAndNamedImport(t *testing.T) {
	rootDir := t.TempDir()
	modulePath := filepath.Join(rootDir, "math.is")
	if err := os.WriteFile(modulePath, []byte(`
const VERSION = "1.0.0"

fn add(a, b) {
    return a + b
}

export add
export VERSION
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	env := object.NewEnvironment()
	env.SetCLIContext(filepath.Join(rootDir, "main.is"), nil)
	result := Eval(parseProgramForTest(t, `
import "./math.is"
import { add, VERSION } from "./math.is"
sum_a = math.add(2, 3)
sum_b = add(4, 5)
version = VERSION
module_version = math.VERSION
`), env)
	env.Wait()

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "sum_a"); got != "5" {
		t.Fatalf("expected sum_a=5, got %s", got)
	}
	if got := testStringValue(t, env, "sum_b"); got != "9" {
		t.Fatalf("expected sum_b=9, got %s", got)
	}
	if got := testStringValue(t, env, "version"); got != "1.0.0" {
		t.Fatalf("expected version=1.0.0, got %s", got)
	}
	if got := testStringValue(t, env, "module_version"); got != "1.0.0" {
		t.Fatalf("expected module_version=1.0.0, got %s", got)
	}
}

func TestModuleImportNestedRelativePath(t *testing.T) {
	rootDir := t.TempDir()
	libDir := filepath.Join(rootDir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib: %v", err)
	}

	if err := os.WriteFile(filepath.Join(libDir, "constants.is"), []byte(`
const BASE = 21
export BASE
`), 0o644); err != nil {
		t.Fatalf("write constants module: %v", err)
	}

	if err := os.WriteFile(filepath.Join(libDir, "math.is"), []byte(`
import { BASE } from "./constants.is"

fn double_base() {
    return BASE * 2
}

export double_base
`), 0o644); err != nil {
		t.Fatalf("write math module: %v", err)
	}

	env := object.NewEnvironment()
	env.SetCLIContext(filepath.Join(rootDir, "main.is"), nil)
	result := Eval(parseProgramForTest(t, `
import "./lib/math.is" as math
answer = math.double_base()
`), env)
	env.Wait()

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "answer"); got != "42" {
		t.Fatalf("expected answer=42, got %s", got)
	}
}

func parseProgramForTest(t *testing.T, input string) *ast.Program {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parser errors: %v", errs)
	}
	return program
}

func testStringValue(t *testing.T, env *object.Environment, name string) string {
	t.Helper()

	value, ok := env.Get(name)
	if !ok {
		t.Fatalf("%s not found in environment", name)
	}
	return value.Inspect()
}

func testArrayValue(t *testing.T, env *object.Environment, name string) *object.Array {
	t.Helper()

	value, ok := env.Get(name)
	if !ok {
		t.Fatalf("%s not found in environment", name)
	}

	array, ok := value.(*object.Array)
	if !ok {
		t.Fatalf("expected %s to be ARRAY, got %T", name, value)
	}
	return array
}

func testHashValue(t *testing.T, env *object.Environment, name string) *object.Hash {
	t.Helper()

	value, ok := env.Get(name)
	if !ok {
		t.Fatalf("%s not found in environment", name)
	}

	hash, ok := value.(*object.Hash)
	if !ok {
		t.Fatalf("expected %s to be HASH, got %T", name, value)
	}
	return hash
}

func testArrayContains(array *object.Array, expected string) bool {
	for _, item := range array.Elements {
		if item.Inspect() == expected {
			return true
		}
	}
	return false
}
