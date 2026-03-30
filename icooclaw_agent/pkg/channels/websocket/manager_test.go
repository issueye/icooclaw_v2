package websocket

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/models"
	"icooclaw/pkg/consts"

	"github.com/go-chi/chi/v5"
)

func TestNewManager_AssignsRouter(t *testing.T) {
	router := chi.NewRouter()
	manager := NewManager(DefaultManagerConfig(), slog.Default(), router, bus.NewMessageBus(bus.DefaultConfig()))
	if manager.router == nil {
		t.Fatal("expected router to be assigned")
	}
}

func TestManagerStart_RequiresRouter(t *testing.T) {
	manager := NewManager(DefaultManagerConfig(), slog.Default(), nil, bus.NewMessageBus(bus.DefaultConfig()))
	if err := manager.Start(context.Background()); err == nil {
		t.Fatal("expected error when router is nil")
	}
}

func TestManagerStart_WithRouterSucceeds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(DefaultManagerConfig(), slog.Default(), chi.NewRouter(), bus.NewMessageBus(bus.DefaultConfig()))
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestBuildSendPayload_Chunk(t *testing.T) {
	eventType, payload := buildSendPayload(models.OutboundMessage{
		SessionID: "session-1",
		Text:      "hello",
		Metadata: map[string]any{
			"event_type": "chunk",
			"reasoning":  "think",
			"iteration":  2,
		},
	})
	if eventType != "chunk" {
		t.Fatalf("eventType = %q, want %q", eventType, "chunk")
	}
	data := payload["data"].(map[string]any)
	if data["content"] != "hello" {
		t.Fatalf("content = %v, want %q", data["content"], "hello")
	}
	if data["reasoning"] != "think" {
		t.Fatalf("reasoning = %v, want %q", data["reasoning"], "think")
	}
}

func TestBuildSendPayload_ToolCall(t *testing.T) {
	eventType, payload := buildSendPayload(models.OutboundMessage{
		SessionID: "session-2",
		Metadata: map[string]any{
			"event_type":   "tool_call",
			"tool_call_id": "call-1",
			"tool_name":    "search",
			"tool_args":    `{"q":"x"}`,
		},
	})
	if eventType != "tool_call" {
		t.Fatalf("eventType = %q, want %q", eventType, "tool_call")
	}
	data := payload["data"].(map[string]any)
	if data["tool_call_id"] != "call-1" {
		t.Fatalf("tool_call_id = %v, want %q", data["tool_call_id"], "call-1")
	}
}

func TestProcessStreamMessage_PublishesInbound(t *testing.T) {
	msgBus := bus.NewMessageBus(bus.DefaultConfig())
	manager := NewManager(DefaultManagerConfig(), slog.Default(), chi.NewRouter(), msgBus)
	client := &Client{
		ID:        "client-1",
		userID:    "user-1",
		sessionID: "session-1",
		manager:   manager,
		logger:    slog.Default(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := manager.ProcessStreamMessage(ctx, client, &ChatMessage{
		SessionID: "session-1",
		Content:   "hello",
		Stream:    true,
	})
	if err != nil {
		t.Fatalf("ProcessStreamMessage() error = %v", err)
	}

	inbound, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message")
	}
	if inbound.Channel != consts.WEBSOCKET {
		t.Fatalf("channel = %q, want %q", inbound.Channel, consts.WEBSOCKET)
	}
	if inbound.Text != "hello" {
		t.Fatalf("text = %q, want %q", inbound.Text, "hello")
	}
}
