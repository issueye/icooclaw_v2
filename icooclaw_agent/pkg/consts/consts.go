package consts

import (
	"fmt"
	"time"
)

type RoleType string

const (
	RoleUser       RoleType = "user"
	RoleAgent      RoleType = "agent"
	RoleSystem     RoleType = "system"
	RoleAssistant  RoleType = "assistant"
	RoleTool       RoleType = "tool"
	RoleToolCall   RoleType = "tool_call"
	RoleToolResult RoleType = "tool_result"
)

func (r RoleType) ToString() string {
	return string(r)
}

func ToRole(role string) RoleType {
	return RoleType(role)
}

// DEFAULT_MODEL_KEY 默认模型键名
const DEFAULT_MODEL_KEY = "agent.default_model"
const SKILL_DEFAULT_MODEL_KEY = "skill.default_model"
const DEFAULT_AGENT_ID_KEY = "agent.default_agent_id"
const EXEC_ENV_KEY = "exec.env"

const SKILL_DIR = "skills"

const DEFAULT_TOOL_ITERATIONS = 30

const (
	DINGTALK  = "dingtalk"
	FEISHU    = "feishu"
	TELEGRAM  = "telegram"
	WEB       = "web"
	WEBSOCKET = "websocket"
	ICOO_CHAT = "icoo_chat"
	QQ        = "qq"

	FEISHU_CN = "飞书"
)

// Default rate limits per channel (messages per second).
var ChannelRateConfig = map[string]float64{
	DINGTALK: 10,
	FEISHU:   10,
	TELEGRAM: 20,
	QQ:       20,
}

const (
	DefaultRateLimit      = 10
	DefaultRetries        = 3
	DefaultRateLimitDelay = 1 * time.Second
	DefaultBaseBackoff    = 500 * time.Millisecond
	DefaultMaxBackoff     = 8 * time.Second
)

// ProviderType represents a provider type.
type ProviderType string

type ProviderProtocol string

const (
	ProviderOpenAI         ProviderType = "openai"
	ProviderAnthropic      ProviderType = "anthropic"
	ProviderDeepSeek       ProviderType = "deepseek"
	ProviderOpenRouter     ProviderType = "openrouter"
	ProviderMiniMax        ProviderType = "minimax"
	ProviderGemini         ProviderType = "gemini"
	ProviderMistral        ProviderType = "mistral"
	ProviderGroq           ProviderType = "groq"
	ProviderAzure          ProviderType = "azure"
	ProviderOllama         ProviderType = "ollama"
	ProviderMoonshot       ProviderType = "moonshot"
	ProviderZhipu          ProviderType = "zhipu"
	ProviderQwen           ProviderType = "qwen"
	ProviderQwenCodingPlan ProviderType = "qwen_coding_plan"
	ProviderSiliconFlow    ProviderType = "siliconflow"
	ProviderGrok           ProviderType = "grok"
)

const (
	ProtocolOpenAI    ProviderProtocol = "openai"
	ProtocolAnthropic ProviderProtocol = "anthropic"
)

func (p ProviderType) ToString() string {
	return string(p)
}

func ToProviderType(providerType string) ProviderType {
	return ProviderType(providerType)
}

func (p ProviderType) String() string {
	return string(p)
}

func (p ProviderProtocol) ToString() string {
	return string(p)
}

func ToProviderProtocol(protocol string) ProviderProtocol {
	return ProviderProtocol(protocol)
}

func (p ProviderProtocol) String() string {
	return string(p)
}

const DEFAULT_AGENT_NAME = "default"
const HookScriptDir = "hooks"
const SummaryMetadataType = "summary"
const DefaultRecentCount = 5
const DefaultHookSummaryMinBatch = 8

// GetSessionKey 生成会话键，格式: channel:sessionID
func GetSessionKey(channel, sessionID string) string {
	return fmt.Sprintf("%s:%s", channel, sessionID)
}

type HOOKType string

const (
	HookGetProvider         HOOKType = "onGetProvider"         // onGetProvider
	HookCreateAgent         HOOKType = "onCreateAgent"         // onCreateAgent
	HookAgentStart          HOOKType = "onAgentStart"          // onAgentStart
	HookAgentEnd            HOOKType = "onAgentEnd"            // onAgentEnd
	HookAgentEndWithSummary HOOKType = "onAgentEndWithSummary" // onAgentEndWithSummary

	HookBuildMessagesBefore HOOKType = "onBuildMessagesBefore" // onBuildMessagesBefore
	HookBuildMessagesAfter  HOOKType = "onBuildMessagesAfter"  // onBuildMessagesAfter
	HookRunLLMBefore        HOOKType = "onRunLLMBefore"        // onRunLLMBefore
	HookRunLLMAfter         HOOKType = "onRunLLMAfter"         // onRunLLMAfter
	HookToolCallBefore      HOOKType = "onToolCallBefore"      // onToolCallBefore
	HookToolCallAfter       HOOKType = "onToolCallAfter"       // onToolCallAfter
	HookToolParseArguments  HOOKType = "onToolParseArguments"  // onToolParseArguments
)

func (h HOOKType) ToString() string {
	return string(h)
}
