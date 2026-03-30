package platforms

import (
	"testing"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
)

func TestResolveAnthropicChatPath_DefaultAnthropicAPI(t *testing.T) {
	got := resolveAnthropicChatPath(&storage.Provider{
		Type:     consts.ProviderAnthropic,
		Protocol: consts.ProtocolAnthropic,
	}, "https://api.anthropic.com/v1")

	if got != "/messages" {
		t.Fatalf("resolveAnthropicChatPath() = %q, want /messages", got)
	}
}

func TestResolveAnthropicChatPath_QianfanCodingEndpoint(t *testing.T) {
	got := resolveAnthropicChatPath(&storage.Provider{
		Name:     "qianfan",
		Type:     consts.ProviderAnthropic,
		Protocol: consts.ProtocolAnthropic,
	}, "https://qianfan.baidubce.com/anthropic/coding")

	if got != "/v1/messages" {
		t.Fatalf("resolveAnthropicChatPath() = %q, want /v1/messages", got)
	}
}

func TestResolveAnthropicChatPath_ConfigOverride(t *testing.T) {
	got := resolveAnthropicChatPath(&storage.Provider{
		Type:     consts.ProviderAnthropic,
		Protocol: consts.ProtocolAnthropic,
		Config:   `{"chat_path":"custom/messages"}`,
	}, "https://qianfan.baidubce.com/anthropic/coding")

	if got != "/custom/messages" {
		t.Fatalf("resolveAnthropicChatPath() = %q, want /custom/messages", got)
	}
}
