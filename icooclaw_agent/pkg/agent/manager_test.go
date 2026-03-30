package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestBuildStreamOutboundMessages_Chunk(t *testing.T) {
	inbound := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "session-1",
	}

	messages := buildStreamOutboundMessages(inbound, react.StreamChunk{
		Content:   "hello",
		Reasoning: "think",
		Iteration: 2,
	})
	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	if messages[0].Metadata[outboundEventTypeKey] != outboundEventChunk {
		t.Fatalf("event type = %v, want %q", messages[0].Metadata[outboundEventTypeKey], outboundEventChunk)
	}
	if messages[0].Text != "hello" {
		t.Fatalf("text = %q, want %q", messages[0].Text, "hello")
	}
	if messages[0].Metadata[outboundReasoningKey] != "think" {
		t.Fatalf("reasoning = %v, want %q", messages[0].Metadata[outboundReasoningKey], "think")
	}
}

func TestBuildStreamOutboundMessages_ToolAndEnd(t *testing.T) {
	inbound := bus.InboundMessage{
		Channel:   consts.ICOO_CHAT,
		SessionID: "session-2",
	}

	messages := buildStreamOutboundMessages(inbound, react.StreamChunk{
		ToolCallID: "call-1",
		ToolName:   "search",
		ToolArgs:   `{"q":"test"}`,
		ToolResult: "ok",
		Done:       true,
		Iteration:  1,
	})
	if len(messages) != 3 {
		t.Fatalf("message count = %d, want 3", len(messages))
	}
	if messages[0].Metadata[outboundEventTypeKey] != outboundEventToolCall {
		t.Fatalf("first event type = %v, want %q", messages[0].Metadata[outboundEventTypeKey], outboundEventToolCall)
	}
	if messages[1].Metadata[outboundEventTypeKey] != outboundEventToolResult {
		t.Fatalf("second event type = %v, want %q", messages[1].Metadata[outboundEventTypeKey], outboundEventToolResult)
	}
	if messages[2].Metadata[outboundEventTypeKey] != outboundEventEnd {
		t.Fatalf("third event type = %v, want %q", messages[2].Metadata[outboundEventTypeKey], outboundEventEnd)
	}
}

func TestBuildStreamOutboundMessages_Error(t *testing.T) {
	inbound := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "session-3",
	}

	messages := buildStreamOutboundMessages(inbound, react.StreamChunk{
		Error:     errors.New("boom"),
		Iteration: 3,
	})
	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	if messages[0].Metadata[outboundEventTypeKey] != outboundEventError {
		t.Fatalf("event type = %v, want %q", messages[0].Metadata[outboundEventTypeKey], outboundEventError)
	}
	if messages[0].Text != "boom" {
		t.Fatalf("text = %q, want %q", messages[0].Text, "boom")
	}
}

func TestEnsureSession_CreatesSession(t *testing.T) {
	store := newTestStorageForAgent(t)
	manager := &AgentManager{storage: store}

	msg := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: "session-create",
		Sender:    bus.SenderInfo{ID: "user-1"},
		Text:      "这是第一条消息，用来生成标题",
		Metadata: map[string]any{
			"agent_id": "agent-1",
		},
	}

	if err := manager.ensureSession(msg); err != nil {
		t.Fatalf("ensureSession() error = %v", err)
	}

	session, err := store.Session().GetBySessionID(consts.WEBSOCKET, "session-create")
	if err != nil {
		t.Fatalf("GetBySessionID() error = %v", err)
	}
	if session.UserID != "user-1" {
		t.Fatalf("user_id = %q, want %q", session.UserID, "user-1")
	}
	if session.AgentID != "agent-1" {
		t.Fatalf("agent_id = %q, want %q", session.AgentID, "agent-1")
	}
	if session.Title == "" {
		t.Fatal("expected title to be generated")
	}
}

func TestEnsureSession_UpdatesMissingFields(t *testing.T) {
	store := newTestStorageForAgent(t)
	if err := store.Session().Save(&storage.Session{
		Model: storage.Model{ID: "session-update"},
		Channel: consts.ICOO_CHAT,
	}); err != nil {
		t.Fatalf("Save() session error = %v", err)
	}

	manager := &AgentManager{storage: store}
	msg := bus.InboundMessage{
		Channel:   consts.ICOO_CHAT,
		SessionID: "session-update",
		Sender:    bus.SenderInfo{ID: "user-2"},
		Text:      "补充标题",
	}

	if err := manager.ensureSession(msg); err != nil {
		t.Fatalf("ensureSession() error = %v", err)
	}

	session, err := store.Session().GetBySessionID(consts.ICOO_CHAT, "session-update")
	if err != nil {
		t.Fatalf("GetBySessionID() error = %v", err)
	}
	if session.UserID != "user-2" {
		t.Fatalf("user_id = %q, want %q", session.UserID, "user-2")
	}
	if session.Title != "补充标题" {
		t.Fatalf("title = %q, want %q", session.Title, "补充标题")
	}
}

func newTestStorageForAgent(t *testing.T) *storage.Storage {
	t.Helper()

	workspaceDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "agent.db")
	store, err := storage.New(workspaceDir, "", dbPath)
	if err != nil {
		t.Fatalf("storage.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}
