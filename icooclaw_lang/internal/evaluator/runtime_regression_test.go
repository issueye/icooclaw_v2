package evaluator

import (
	"runtime"
	"testing"
	"time"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestHTTPServerRepeatedStartStopDoesNotLeakGoroutines(t *testing.T) {
	baseline := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		_, result := evalSource(t, `
server = http.server.new()
server.route("/health", "ok")
addr = server.start("127.0.0.1:0")
resp = http.client.get(server.url("/health"))
server.stop()
`)
		if object.IsError(result) {
			t.Fatalf("iteration %d unexpected eval error: %s", i, result.Inspect())
		}
	}

	time.Sleep(30 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > baseline+3 {
		t.Fatalf("possible http server goroutine leak: baseline=%d after=%d", baseline, after)
	}
}

func TestGoStatementBurstWithSleepDoesNotLeakGoroutines(t *testing.T) {
	baseline := runtime.NumGoroutine()

	_, result := evalSource(t, `
fn sleeper(v) {
    time.sleep_ms(1)
    return v
}

for i in range(100) {
    go sleeper(i)
}
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	time.Sleep(40 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > baseline+3 {
		t.Fatalf("possible goroutine leak after go burst: baseline=%d after=%d", baseline, after)
	}
}

func TestRepeatedJSONRoundTripUnderLoad(t *testing.T) {
	_, result := evalSource(t, `
payload = {
    "name": "icooclaw",
    "values": [1, 2, 3],
    "nested": {"ok": true},
}

for i in range(200) {
    text = json.stringify(payload)
    parsed = json.parse(text)
    if parsed.nested.ok != true {
        panic_value = "bad"
    }
}
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}
}
