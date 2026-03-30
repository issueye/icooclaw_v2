package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

type agentToolBase struct {
	storage *storage.Storage
}

func newAgentToolBase(st *storage.Storage) agentToolBase {
	return agentToolBase{storage: st}
}

func (b agentToolBase) ensureStorage() error {
	if b.storage == nil || b.storage.Agent() == nil {
		return fmt.Errorf("agent storage is not configured")
	}
	return nil
}

func marshalAgentResult(v any) *tools.Result {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("marshal result failed: %w", err)}
	}
	return &tools.Result{Success: true, Content: string(data)}
}

func agentStringArg(args map[string]any, key string) string {
	if value, ok := args[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func agentTypeArg(args map[string]any, key string) (string, error) {
	value := agentStringArg(args, key)
	if !storage.IsValidAgentType(value) {
		return "", fmt.Errorf("type must be master or subagent")
	}
	return storage.NormalizeAgentType(value), nil
}

func boolArgPtr(args map[string]any, key string) *bool {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func metadataArg(args map[string]any, key string) (map[string]any, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return map[string]any{}, nil
	}

	switch typed := value.(type) {
	case map[string]any:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return map[string]any{}, nil
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("metadata must be a JSON object: %w", err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("metadata must be an object or JSON string")
	}
}

type ListAgentsTool struct{ agentToolBase }

func NewListAgentsTool(st *storage.Storage) *ListAgentsTool {
	return &ListAgentsTool{agentToolBase: newAgentToolBase(st)}
}

func (t *ListAgentsTool) Name() string { return "list_agents" }
func (t *ListAgentsTool) Description() string {
	return "查询智能体列表，可按启用状态、类型和关键字过滤。"
}
func (t *ListAgentsTool) Parameters() map[string]any {
	return map[string]any{
		"enabled": map[string]any{
			"type":        "boolean",
			"description": "可选，按是否启用过滤。",
		},
		"keyword": map[string]any{
			"type":        "string",
			"description": "可选，按名称或描述模糊搜索。",
		},
		"type": map[string]any{
			"type":        "string",
			"description": "可选，智能体类型：master 或 subagent。",
			"enum":        []string{storage.AgentTypeMaster, storage.AgentTypeSubAgent},
		},
	}
}
func (t *ListAgentsTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	query := &storage.QueryAgent{
		KeyWord: agentStringArg(args, "keyword"),
		Enabled: boolArgPtr(args, "enabled"),
		Type:    agentStringArg(args, "type"),
	}

	page, err := t.storage.Agent().Page(query)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("list agents failed: %w", err)}
	}
	return marshalAgentResult(page.Records)
}

type GetAgentTool struct{ agentToolBase }

func NewGetAgentTool(st *storage.Storage) *GetAgentTool {
	return &GetAgentTool{agentToolBase: newAgentToolBase(st)}
}

func (t *GetAgentTool) Name() string        { return "get_agent" }
func (t *GetAgentTool) Description() string { return "按 ID 或名称获取单个智能体详情。" }
func (t *GetAgentTool) Parameters() map[string]any {
	return map[string]any{
		"id": map[string]any{
			"type":        "string",
			"description": "智能体 ID。和 name 二选一。",
		},
		"name": map[string]any{
			"type":        "string",
			"description": "智能体名称。和 id 二选一。",
		},
	}
}
func (t *GetAgentTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	id := agentStringArg(args, "id")
	name := agentStringArg(args, "name")

	var (
		agentInfo *storage.Agent
		err       error
	)

	switch {
	case id != "":
		agentInfo, err = t.storage.Agent().GetByID(id)
	case name != "":
		agentInfo, err = t.storage.Agent().GetByName(name)
	default:
		return &tools.Result{Success: false, Error: fmt.Errorf("either id or name is required")}
	}
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("get agent failed: %w", err)}
	}
	return marshalAgentResult(agentInfo)
}

type CreateAgentTool struct{ agentToolBase }

func NewCreateAgentTool(st *storage.Storage) *CreateAgentTool {
	return &CreateAgentTool{agentToolBase: newAgentToolBase(st)}
}

func (t *CreateAgentTool) Name() string { return "create_agent" }
func (t *CreateAgentTool) Description() string {
	return "创建一个新的智能体，包含名称、类型、描述、系统提示词和元数据。"
}
func (t *CreateAgentTool) Parameters() map[string]any {
	return map[string]any{
		"name": map[string]any{
			"type":        "string",
			"description": "智能体名称。",
			"required":    true,
		},
		"description": map[string]any{
			"type":        "string",
			"description": "智能体描述。",
		},
		"type": map[string]any{
			"type":        "string",
			"description": "智能体类型，默认 master。",
			"enum":        []string{storage.AgentTypeMaster, storage.AgentTypeSubAgent},
		},
		"system_prompt": map[string]any{
			"type":        "string",
			"description": "额外系统提示词。",
		},
		"enabled": map[string]any{
			"type":        "boolean",
			"description": "是否启用，默认 true。",
		},
		"metadata": map[string]any{
			"type":        "object",
			"description": "可选元数据对象，也可传 JSON 字符串。",
		},
	}
}
func (t *CreateAgentTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	name := agentStringArg(args, "name")
	if name == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("name is required")}
	}

	metadata, err := metadataArg(args, "metadata")
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}
	agentType, err := agentTypeArg(args, "type")
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	enabled := true
	if v := boolArgPtr(args, "enabled"); v != nil {
		enabled = *v
	}

	agentInfo := &storage.Agent{
		Name:         name,
		Type:         agentType,
		Description:  agentStringArg(args, "description"),
		SystemPrompt: agentStringArg(args, "system_prompt"),
		Enabled:      enabled,
		Metadata:     metadata,
	}
	if err := t.storage.Agent().Save(agentInfo); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("create agent failed: %w", err)}
	}

	saved, err := t.storage.Agent().GetByName(name)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("load created agent failed: %w", err)}
	}
	return marshalAgentResult(saved)
}

type UpdateAgentTool struct{ agentToolBase }

func NewUpdateAgentTool(st *storage.Storage) *UpdateAgentTool {
	return &UpdateAgentTool{agentToolBase: newAgentToolBase(st)}
}

func (t *UpdateAgentTool) Name() string        { return "update_agent" }
func (t *UpdateAgentTool) Description() string { return "更新已有智能体，id 为必填。" }
func (t *UpdateAgentTool) Parameters() map[string]any {
	return map[string]any{
		"id": map[string]any{
			"type":        "string",
			"description": "智能体 ID。",
			"required":    true,
		},
		"name": map[string]any{
			"type":        "string",
			"description": "新的智能体名称。",
		},
		"description": map[string]any{
			"type":        "string",
			"description": "新的描述。",
		},
		"type": map[string]any{
			"type":        "string",
			"description": "新的智能体类型。",
			"enum":        []string{storage.AgentTypeMaster, storage.AgentTypeSubAgent},
		},
		"system_prompt": map[string]any{
			"type":        "string",
			"description": "新的系统提示词。",
		},
		"enabled": map[string]any{
			"type":        "boolean",
			"description": "新的启用状态。",
		},
		"metadata": map[string]any{
			"type":        "object",
			"description": "新的元数据对象，也可传 JSON 字符串。",
		},
	}
}
func (t *UpdateAgentTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	id := agentStringArg(args, "id")
	if id == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("id is required")}
	}

	agentInfo, err := t.storage.Agent().GetByID(id)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("load agent failed: %w", err)}
	}

	if name := agentStringArg(args, "name"); name != "" {
		agentInfo.Name = name
	}
	if _, ok := args["type"]; ok {
		agentType, err := agentTypeArg(args, "type")
		if err != nil {
			return &tools.Result{Success: false, Error: err}
		}
		agentInfo.Type = agentType
	}
	if _, ok := args["description"]; ok {
		agentInfo.Description = agentStringArg(args, "description")
	}
	if _, ok := args["system_prompt"]; ok {
		agentInfo.SystemPrompt = agentStringArg(args, "system_prompt")
	}
	if v := boolArgPtr(args, "enabled"); v != nil {
		agentInfo.Enabled = *v
	}
	if _, ok := args["metadata"]; ok {
		metadata, err := metadataArg(args, "metadata")
		if err != nil {
			return &tools.Result{Success: false, Error: err}
		}
		agentInfo.Metadata = metadata
	}

	if err := t.storage.Agent().Save(agentInfo); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("update agent failed: %w", err)}
	}
	return marshalAgentResult(agentInfo)
}

type DeleteAgentTool struct{ agentToolBase }

func NewDeleteAgentTool(st *storage.Storage) *DeleteAgentTool {
	return &DeleteAgentTool{agentToolBase: newAgentToolBase(st)}
}

func (t *DeleteAgentTool) Name() string        { return "delete_agent" }
func (t *DeleteAgentTool) Description() string { return "按 ID 删除智能体。" }
func (t *DeleteAgentTool) Parameters() map[string]any {
	return map[string]any{
		"id": map[string]any{
			"type":        "string",
			"description": "要删除的智能体 ID。",
			"required":    true,
		},
	}
}
func (t *DeleteAgentTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if err := t.ensureStorage(); err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	id := agentStringArg(args, "id")
	if id == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("id is required")}
	}

	if err := t.storage.Agent().Delete(id); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("delete agent failed: %w", err)}
	}
	return marshalAgentResult(map[string]any{
		"id":      id,
		"deleted": true,
	})
}
