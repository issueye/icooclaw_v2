package evaluator

import (
	"encoding/json"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestWebSocketServerAndClient(t *testing.T) {
	env, result := evalSource(t, `
fn echo(req, socket) {
    message = socket.read()
    socket.send("echo:" + message)
}

server = websocket.server.new()
server.handle("/echo", echo)
before_running = server.is_running()
addr = server.start("127.0.0.1:0")
client = websocket.client.connect(server.url("/echo"))
client.send("hello")
reply = client.read()
client.close()
stats = server.stats()
server.stop()
after_running = server.is_running()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "before_running"); got != "false" {
		t.Fatalf("expected before_running=false, got %s", got)
	}
	if got := testStringValue(t, env, "reply"); got != "echo:hello" {
		t.Fatalf("expected reply=echo:hello, got %s", got)
	}
	stats := testHashValue(t, env, "stats")
	if stats.Pairs["handler_count"].Value.Inspect() != "1" {
		t.Fatalf("expected handler_count=1, got %s", stats.Pairs["handler_count"].Value.Inspect())
	}
	if stats.Pairs["request_count"].Value.Inspect() != "1" {
		t.Fatalf("expected request_count=1, got %s", stats.Pairs["request_count"].Value.Inspect())
	}
	if stats.Pairs["connection_count"].Value.Inspect() != "1" {
		t.Fatalf("expected connection_count=1, got %s", stats.Pairs["connection_count"].Value.Inspect())
	}
	if got := testStringValue(t, env, "after_running"); got != "false" {
		t.Fatalf("expected after_running=false, got %s", got)
	}
}

func TestWebSocketReadMessageAndJSONReply(t *testing.T) {
	env, result := evalSource(t, `
fn reflect(req, socket) {
    incoming = socket.read_message()
    return {"type": incoming.type, "data": incoming.data}
}

server = websocket.server.new()
server.handle("/reflect", reflect)
addr = server.start("127.0.0.1:0")
client = websocket.client.connect(server.url("/reflect"))
client.send("world")
reply_text = client.read()
reply = json.parse(reply_text)
client.close()
server.stop()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	reply := testHashValue(t, env, "reply")
	if reply.Pairs["type"].Value.Inspect() != "text" {
		t.Fatalf("expected reply.type=text, got %s", reply.Pairs["type"].Value.Inspect())
	}
	if reply.Pairs["data"].Value.Inspect() != "world" {
		t.Fatalf("expected reply.data=world, got %s", reply.Pairs["data"].Value.Inspect())
	}
}

func TestSSEServerAndClient(t *testing.T) {
	env, result := evalSource(t, `
fn events(req, stream) {
    stream.send("hello")
    stream.send_event("update", "world")
    stream.send_json({"ok": true})
}

server = sse.server.new()
server.handle("/events", events)
before_running = server.is_running()
addr = server.start("127.0.0.1:0")
client = sse.client.connect(server.url("/events"))
first = client.read()
second = client.read()
third = client.read()
fourth = client.read()
client_closed_before = client.is_closed()
client.close()
stats = server.stats()
server.stop()
after_running = server.is_running()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	first := testHashValue(t, env, "first")
	if first.Pairs["event"].Value.Inspect() != "message" || first.Pairs["data"].Value.Inspect() != "hello" {
		t.Fatalf("unexpected first event: %s", first.Inspect())
	}

	second := testHashValue(t, env, "second")
	if second.Pairs["event"].Value.Inspect() != "update" || second.Pairs["data"].Value.Inspect() != "world" {
		t.Fatalf("unexpected second event: %s", second.Inspect())
	}

	third := testHashValue(t, env, "third")
	if third.Pairs["event"].Value.Inspect() != "json" {
		t.Fatalf("expected third.event=json, got %s", third.Pairs["event"].Value.Inspect())
	}
	decoded := evalSourceJSON(t, third.Pairs["data"].Value.Inspect())
	if decoded.Pairs["ok"].Value.Inspect() != "true" {
		t.Fatalf("expected decoded.ok=true, got %s", decoded.Pairs["ok"].Value.Inspect())
	}

	if got := testStringValue(t, env, "fourth"); got != "null" {
		t.Fatalf("expected fourth=null, got %s", got)
	}
	if got := testStringValue(t, env, "client_closed_before"); got != "true" {
		t.Fatalf("expected client_closed_before=true, got %s", got)
	}

	stats := testHashValue(t, env, "stats")
	if stats.Pairs["handler_count"].Value.Inspect() != "1" {
		t.Fatalf("expected handler_count=1, got %s", stats.Pairs["handler_count"].Value.Inspect())
	}
	if stats.Pairs["request_count"].Value.Inspect() != "1" {
		t.Fatalf("expected request_count=1, got %s", stats.Pairs["request_count"].Value.Inspect())
	}
	if got := testStringValue(t, env, "after_running"); got != "false" {
		t.Fatalf("expected after_running=false, got %s", got)
	}
}

func evalSourceJSON(t *testing.T, payload string) *object.Hash {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unexpected json decode error: %v", err)
	}
	return objectFromMap(decoded)
}

func objectFromMap(input map[string]any) *object.Hash {
	values := make(map[string]object.Object, len(input))
	for key, value := range input {
		switch v := value.(type) {
		case bool:
			values[key] = &object.Boolean{Value: v}
		case string:
			values[key] = &object.String{Value: v}
		case float64:
			values[key] = &object.Integer{Value: int64(v)}
		default:
			values[key] = &object.String{Value: payloadString(v)}
		}
	}
	return hashForTest(values)
}

func hashForTest(values map[string]object.Object) *object.Hash {
	pairs := make(map[string]object.HashPair, len(values))
	for key, value := range values {
		keyObj := &object.String{Value: key}
		pairs[key] = object.HashPair{Key: keyObj, Value: value}
	}
	return &object.Hash{Pairs: pairs}
}

func payloadString(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}
