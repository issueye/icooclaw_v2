package handlers

import "testing"

func TestNormalizeTaskChannel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "websocket stays canonical", input: "websocket", want: "websocket"},
		{name: "feishu chinese alias", input: "飞书", want: "feishu"},
		{name: "dingtalk chinese alias", input: "钉钉", want: "dingtalk"},
		{name: "trim unknown channel", input: "  webhook  ", want: "webhook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTaskChannel(tt.input); got != tt.want {
				t.Fatalf("normalizeTaskChannel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
