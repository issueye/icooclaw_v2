// Package providers provides LLM provider abstraction for icooclaw.
package providers

import "icooclaw/pkg/providers/protocol"

type ChatMessage = protocol.ChatMessage
type ChatRequest = protocol.ChatRequest
type Function = protocol.Function
type Tool = protocol.Tool
type ToolCall = protocol.ToolCall
type ChatResponse = protocol.ChatResponse
type Usage = protocol.Usage
type StreamCallback = protocol.StreamCallback
type Provider = protocol.Provider
