package icoo_chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"

	"github.com/gorilla/websocket"
)

func TestChannel_ReceivesInboundAndSendsReply(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	received := make(chan map[string]any, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("app_id") != "app-1" || r.URL.Query().Get("app_secret") != "secret-1" {
			t.Fatalf("unexpected credentials: %s", r.URL.RawQuery)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()

		if err := conn.WriteJSON(map[string]any{"type": "connected"}); err != nil {
			t.Fatalf("write connected: %v", err)
		}
		if err := conn.WriteJSON(map[string]any{
			"type": "message",
			"data": map[string]any{
				"message_id":        "msg-1",
				"session_id":        "session-1",
				"content":           "hello",
				"from_user_id":      "user-1",
				"from_device_id":    "device-1",
				"conversation_type": "c2c",
				"metadata": map[string]any{
					"source": "app",
				},
			},
		}); err != nil {
			t.Fatalf("write message: %v", err)
		}

		var payload map[string]any
		if err := conn.ReadJSON(&payload); err != nil {
			t.Fatalf("read reply: %v", err)
		}
		received <- payload
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	cfg := `{"endpoint":"` + wsURL + `","app_id":"app-1","app_secret":"secret-1"}`
	msgBus := bus.NewMessageBus(bus.DefaultConfig())
	ch, err := New(nil, msgBus, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ch.Stop(context.Background())

	inbound, ok := msgBus.ConsumeInbound(context.Background())
	if !ok {
		t.Fatal("expected inbound message")
	}
	if inbound.Channel != consts.ICOO_CHAT {
		t.Fatalf("channel = %q, want %q", inbound.Channel, consts.ICOO_CHAT)
	}
	if inbound.SessionID != "session-1" || inbound.Text != "hello" {
		t.Fatalf("unexpected inbound: %#v", inbound)
	}

	sendCtx, sendCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer sendCancel()
	if err := ch.Send(sendCtx, models.OutboundMessage{
		Channel:   consts.ICOO_CHAT,
		SessionID: "session-1",
		Text:      "world",
	}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case payload := <-received:
		if payload["type"] != "reply" {
			t.Fatalf("payload type = %v, want reply", payload["type"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for reply payload")
	}
}

func TestChannel_SendsStreamChunkAndEnd(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	received := make(chan map[string]any, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()

		if err := conn.WriteJSON(map[string]any{"type": "connected"}); err != nil {
			t.Fatalf("write connected: %v", err)
		}

		for i := 0; i < 2; i++ {
			var payload map[string]any
			if err := conn.ReadJSON(&payload); err != nil {
				t.Fatalf("read payload %d: %v", i, err)
			}
			received <- payload
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	cfg := `{"endpoint":"` + wsURL + `","app_id":"app-1","app_secret":"secret-1"}`
	msgBus := bus.NewMessageBus(bus.DefaultConfig())
	channel, err := New(nil, msgBus, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := channel.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer channel.Stop(context.Background())

	sendCtx, sendCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer sendCancel()

	if err := channel.Send(sendCtx, models.OutboundMessage{
		Channel:   consts.ICOO_CHAT,
		SessionID: "session-stream",
		Text:      "partial",
		Metadata: map[string]any{
			"event_type": "chunk",
			"reasoning":  "thinking",
		},
	}); err != nil {
		t.Fatalf("Send(chunk) error = %v", err)
	}

	if err := channel.Send(sendCtx, models.OutboundMessage{
		Channel:   consts.ICOO_CHAT,
		SessionID: "session-stream",
		Metadata: map[string]any{
			"event_type": "end",
		},
	}); err != nil {
		t.Fatalf("Send(end) error = %v", err)
	}

	var first, second map[string]any
	select {
	case first = <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for chunk payload")
	}
	select {
	case second = <-received:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for end payload")
	}

	if first["type"] != "chunk" {
		t.Fatalf("first payload type = %v, want chunk", first["type"])
	}
	firstData := first["data"].(map[string]any)
	if firstData["content"] != "partial" {
		t.Fatalf("chunk content = %v, want partial", firstData["content"])
	}
	if firstData["reasoning"] != "thinking" {
		t.Fatalf("chunk reasoning = %v, want thinking", firstData["reasoning"])
	}
	if second["type"] != "end" {
		t.Fatalf("second payload type = %v, want end", second["type"])
	}
}
