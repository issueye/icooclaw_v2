package evaluator

import (
	"fmt"
	"os"
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

func TestTypeOfBuiltinMatchesType(t *testing.T) {
	env, result := evalSource(t, `
kind = type_of(42)
same = type(42) == type_of(42)
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

	sameValue, ok := env.Get("same")
	if !ok {
		t.Fatal("same not found in environment")
	}

	boolValue, ok := sameValue.(*object.Boolean)
	if !ok || !boolValue.Value {
		t.Fatalf("expected same=true, got %s", sameValue.Inspect())
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

func TestHTTPServerAdvancedRoutes(t *testing.T) {
	filePath := filepath.ToSlash(filepath.Join(t.TempDir(), "served.txt"))
	if err := os.WriteFile(filepath.FromSlash(filePath), []byte("served file"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	env, result := evalSource(t, fmt.Sprintf(`
server = http.server.new()
server.route("GET", "/hello", "world")
server.route_response("POST", "/hello", {"status_code": 201, "body": "created", "headers": {"X-Test": "yes"}})
server.route_file("/file", "%s")
server.not_found({"status_code": 418, "body": "missing"})
addr = server.start("127.0.0.1:0")
get_resp = http.client.get(server.url("/hello"))
post_resp = http.client.request("POST", server.url("/hello"), null)
file_resp = http.client.get(server.url("/file"))
missing_resp = http.client.get(server.url("/missing"))
stats = server.stats()
server.stop()
`, filePath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	getResp, _ := env.Get("get_resp")
	if getResp.(*object.Hash).Pairs["body"].Value.Inspect() != "world" {
		t.Fatalf("expected get body=world, got %s", getResp.Inspect())
	}

	postResp, _ := env.Get("post_resp")
	postHash := postResp.(*object.Hash)
	if postHash.Pairs["status_code"].Value.Inspect() != "201" {
		t.Fatalf("expected post status=201, got %s", postHash.Pairs["status_code"].Value.Inspect())
	}
	headerArray := postHash.Pairs["headers"].Value.(*object.Hash).Pairs["X-Test"].Value.(*object.Array)
	if headerArray.Elements[0].Inspect() != "yes" {
		t.Fatalf("expected X-Test=yes, got %s", headerArray.Elements[0].Inspect())
	}

	fileResp, _ := env.Get("file_resp")
	if fileResp.(*object.Hash).Pairs["body"].Value.Inspect() != "served file" {
		t.Fatalf("expected file body=served file, got %s", fileResp.(*object.Hash).Pairs["body"].Value.Inspect())
	}

	missingResp, _ := env.Get("missing_resp")
	if missingResp.(*object.Hash).Pairs["status_code"].Value.Inspect() != "418" {
		t.Fatalf("expected missing status=418, got %s", missingResp.(*object.Hash).Pairs["status_code"].Value.Inspect())
	}

	stats, _ := env.Get("stats")
	if stats.(*object.Hash).Pairs["request_count"].Value.Inspect() != "4" {
		t.Fatalf("expected request_count=4, got %s", stats.(*object.Hash).Pairs["request_count"].Value.Inspect())
	}
}

func TestHTTPServerScriptHandler(t *testing.T) {
	env, result := evalSource(t, `
fn greet(req) {
    if req.method == "POST" {
        return {
            "status_code": 202,
            "body": "hello:" + req.body,
            "headers": {"X-Mode": "post"},
        }
    }

    return {
        "message": "hello " + req.query.name,
        "path": req.path,
    }
}

server = http.server.new()
server.handle("/greet", greet)
addr = server.start("127.0.0.1:0")
get_resp = http.client.get(server.url("/greet?name=icooclaw"))
post_resp = http.client.request("POST", server.url("/greet"), "world")
stats_before_stop = server.stats()
server.stop()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	getResp, _ := env.Get("get_resp")
	getHash := getResp.(*object.Hash)
	if getHash.Pairs["status_code"].Value.Inspect() != "200" {
		t.Fatalf("expected get status=200, got %s", getHash.Pairs["status_code"].Value.Inspect())
	}
	if getHash.Pairs["body"].Value.Inspect() != `{"message":"hello icooclaw","path":"/greet"}` {
		t.Fatalf("unexpected get body: %s", getHash.Pairs["body"].Value.Inspect())
	}

	postResp, _ := env.Get("post_resp")
	postHash := postResp.(*object.Hash)
	if postHash.Pairs["status_code"].Value.Inspect() != "202" {
		t.Fatalf("expected post status=202, got %s", postHash.Pairs["status_code"].Value.Inspect())
	}
	if postHash.Pairs["body"].Value.Inspect() != "hello:world" {
		t.Fatalf("unexpected post body: %s", postHash.Pairs["body"].Value.Inspect())
	}
	postHeaders := postHash.Pairs["headers"].Value.(*object.Hash)
	if postHeaders.Pairs["X-Mode"].Value.(*object.Array).Elements[0].Inspect() != "post" {
		t.Fatalf("expected X-Mode=post, got %s", postHeaders.Pairs["X-Mode"].Value.Inspect())
	}

	stats, _ := env.Get("stats_before_stop")
	if stats.(*object.Hash).Pairs["request_count"].Value.Inspect() != "2" {
		t.Fatalf("expected request_count=2, got %s", stats.(*object.Hash).Pairs["request_count"].Value.Inspect())
	}
}

func TestJSONLibraryEncodeDecode(t *testing.T) {
	env, result := evalSource(t, `
payload = {"name": "alice", "count": 2, "items": [1, true, null]}
encoded = json.stringify(payload)
decoded = json.parse(encoded)
name = decoded.name
count = decoded.count
second = decoded.items[1]
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	name, _ := env.Get("name")
	if name.Inspect() != "alice" {
		t.Fatalf("expected name=alice, got %s", name.Inspect())
	}

	count, _ := env.Get("count")
	if count.Inspect() != "2" {
		t.Fatalf("expected count=2, got %s", count.Inspect())
	}

	second, _ := env.Get("second")
	if second.Inspect() != "true" {
		t.Fatalf("expected second=true, got %s", second.Inspect())
	}
}

func TestTimeAndOSLibraries(t *testing.T) {
	env, result := evalSource(t, `
os.setenv("ICOOCLAW_TEST_ENV", "ok")
cwd = os.cwd()
value = os.getenv("ICOOCLAW_TEST_ENV")
pid = os.pid()
now = time.now()
before = time.now_unix_ms()
time.sleep_ms(5)
after = time.now_unix_ms()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	value, _ := env.Get("value")
	if value.Inspect() != "ok" {
		t.Fatalf("expected env value=ok, got %s", value.Inspect())
	}

	cwd, _ := env.Get("cwd")
	if cwd.Inspect() == "" {
		t.Fatal("expected non-empty cwd")
	}

	pid, _ := env.Get("pid")
	if pid.Inspect() == "0" {
		t.Fatal("expected non-zero pid")
	}

	now, _ := env.Get("now")
	nowHash, ok := now.(*object.Hash)
	if !ok {
		t.Fatalf("expected now hash, got %T", now)
	}
	if _, ok := nowHash.Pairs["rfc_3339"]; !ok {
		t.Fatal("expected now.rfc_3339 field")
	}

	before, _ := env.Get("before")
	after, _ := env.Get("after")
	if !evalComparison(after, before, ">=", 0).(*object.Boolean).Value {
		t.Fatalf("expected after >= before, got before=%s after=%s", before.Inspect(), after.Inspect())
	}
}

func TestJSONLibraryOldAliasesRemoved(t *testing.T) {
	_, result := evalSource(t, `
payload = {"name": "alice"}
json.encode(payload)
`)

	if !object.IsError(result) {
		t.Fatalf("expected alias removal error, got %#v", result)
	}
}

func TestPathLibrary(t *testing.T) {
	env, result := evalSource(t, `
joined = path.join("a", "b", "c.txt")
base = path.base(joined)
ext = path.ext(joined)
dir = path.dir(joined)
cleaned = path.clean("a/./b/../c.txt")
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	joined, _ := env.Get("joined")
	if joined.Inspect() != filepath.Join("a", "b", "c.txt") {
		t.Fatalf("expected joined=%s, got %s", filepath.Join("a", "b", "c.txt"), joined.Inspect())
	}

	base, _ := env.Get("base")
	if base.Inspect() != "c.txt" {
		t.Fatalf("expected base=c.txt, got %s", base.Inspect())
	}

	ext, _ := env.Get("ext")
	if ext.Inspect() != ".txt" {
		t.Fatalf("expected ext=.txt, got %s", ext.Inspect())
	}

	dir, _ := env.Get("dir")
	if dir.Inspect() != filepath.Join("a", "b") {
		t.Fatalf("expected dir=%s, got %s", filepath.Join("a", "b"), dir.Inspect())
	}

	cleaned, _ := env.Get("cleaned")
	if cleaned.Inspect() != filepath.Clean("a/./b/../c.txt") {
		t.Fatalf("expected cleaned=%s, got %s", filepath.Clean("a/./b/../c.txt"), cleaned.Inspect())
	}
}

func TestCryptoLibrary(t *testing.T) {
	env, result := evalSource(t, `
md5_sum = crypto.md5("hello")
sha_1_sum = crypto.sha_1("hello")
sha_256_sum = crypto.sha_256("hello")
encoded = crypto.base_64_encode("hello")
decoded = crypto.base_64_decode(encoded)
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	md5Sum, _ := env.Get("md5_sum")
	if md5Sum.Inspect() != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("unexpected md5: %s", md5Sum.Inspect())
	}

	sha1Sum, _ := env.Get("sha_1_sum")
	if sha1Sum.Inspect() != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("unexpected sha_1: %s", sha1Sum.Inspect())
	}

	sha256Sum, _ := env.Get("sha_256_sum")
	if sha256Sum.Inspect() != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected sha_256: %s", sha256Sum.Inspect())
	}

	decoded, _ := env.Get("decoded")
	if decoded.Inspect() != "hello" {
		t.Fatalf("expected decoded=hello, got %s", decoded.Inspect())
	}
}
