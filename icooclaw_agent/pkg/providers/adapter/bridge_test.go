package adapter

import (
	"testing"

	"icooclaw/pkg/providers/protocol"
)

func TestToAdapterRequest_PreservesRawSource(t *testing.T) {
	req := protocol.ChatRequest{
		Model: "gpt-4o",
		Messages: []protocol.ChatMessage{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	}

	adapted := ToAdapterRequest(req)
	if adapted.Source != "providers" {
		t.Fatalf("source = %q, want %q", adapted.Source, "providers")
	}
	if len(adapted.Raw) == 0 {
		t.Fatal("expected request raw payload")
	}
	if len(adapted.Messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(adapted.Messages))
	}
	if adapted.Messages[0].Source != "providers" {
		t.Fatalf("message source = %q, want %q", adapted.Messages[0].Source, "providers")
	}
	if len(adapted.Messages[0].Raw) == 0 {
		t.Fatal("expected message raw payload")
	}
}
