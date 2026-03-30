package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"

	"icooclaw/pkg/providers"
	"icooclaw/pkg/tools"
)

type Runtime struct {
	parser    *SkillParser
	loader    Loader
	provider  providers.Provider
	modelName string
	logger    *slog.Logger
	tools     *tools.Registry
}

type RuntimeConfig struct {
	Loader    Loader
	Provider  providers.Provider
	ModelName string
	Logger    *slog.Logger
	Tools     *tools.Registry
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Runtime{
		parser:    NewParser(),
		loader:    cfg.Loader,
		provider:  cfg.Provider,
		modelName: cfg.ModelName,
		logger:    cfg.Logger,
		tools:     cfg.Tools,
	}
}

type ExecutionRequest struct {
	SkillName string
	Task      string
	Context   map[string]any
}

type ExecutionResult struct {
	Success     bool
	SkillName   string
	DisplayName string
	Output      string
	Error       error
}

func (r *Runtime) Execute(ctx context.Context, req ExecutionRequest) *ExecutionResult {
	info, err := r.loader.LoadInfo(ctx, req.SkillName)
	if err != nil {
		return &ExecutionResult{
			Success:   false,
			SkillName: req.SkillName,
			Error:     fmt.Errorf("failed to load skill %s: %w", req.SkillName, err),
		}
	}
	displayName := displaySkillName(info)

	r.logger.Info("Executing skill", "name", req.SkillName, "task", req.Task)

	systemPrompt := r.buildSystemPrompt(info, req.Context)
	userMessage := r.buildUserMessage(req.Task, req.Context)

	currentMessages := []providers.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	var lastToolResult string
	var lastToolSignature string

	for iteration := 0; iteration < 8; iteration++ {
		req := providers.ChatRequest{
			Model:    r.modelName,
			Messages: currentMessages,
		}
		if r.tools != nil {
			toolDefs := r.tools.ToProviderDefs()
			if len(toolDefs) > 0 {
				req.Tools = r.convertToolDefinitions(toolDefs)
			}
		}

		chatResp, err := r.provider.Chat(ctx, req)
		if err != nil {
			return &ExecutionResult{
				Success:     false,
				SkillName:   info.Name,
				DisplayName: displayName,
				Error:       fmt.Errorf("LLM call failed: %w", err),
			}
		}

		if len(chatResp.ToolCalls) == 0 {
			return &ExecutionResult{
				Success:     true,
				SkillName:   info.Name,
				DisplayName: displayName,
				Output:      chatResp.Content,
			}
		}

		currentSignature := toolCallSignature(chatResp.ToolCalls)
		if currentSignature != "" && currentSignature == lastToolSignature && lastToolResult != "" {
			return &ExecutionResult{
				Success:     true,
				SkillName:   info.Name,
				DisplayName: displayName,
				Output:      lastToolResult,
			}
		}
		lastToolSignature = currentSignature

		currentMessages = append(currentMessages, providers.ChatMessage{
			Role:      "assistant",
			Content:   chatResp.Content,
			ToolCalls: chatResp.ToolCalls,
		})

		for _, tc := range chatResp.ToolCalls {
			toolResult := r.executeToolCall(ctx, tc)
			if strings.TrimSpace(toolResult) != "" {
				lastToolResult = toolResult
			}
			currentMessages = append(currentMessages, providers.ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	if strings.TrimSpace(lastToolResult) != "" {
		return &ExecutionResult{
			Success:     true,
			SkillName:   info.Name,
			DisplayName: displayName,
			Output:      lastToolResult,
		}
	}

	return &ExecutionResult{
		Success:     false,
		SkillName:   info.Name,
		DisplayName: displayName,
		Error:       fmt.Errorf("skill execution reached max tool iterations"),
	}
}

func (r *Runtime) buildSystemPrompt(skill *Info, ctx map[string]any) string {
	var sb strings.Builder
	sb.WriteString("# Skill: ")
	sb.WriteString(displaySkillName(skill))
	sb.WriteString("\n\n")
	sb.WriteString(skill.Description)
	sb.WriteString("\n\n")
	sb.WriteString("## Instructions\n")
	sb.WriteString(skill.Content)
	sb.WriteString("\n\n")
	if ctx != nil {
		if notes, ok := ctx["skill_notes"].(string); ok && notes != "" {
			sb.WriteString("## Additional Context\n")
			sb.WriteString(notes)
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("## Runtime Environment\n")
	sb.WriteString("Operating system: ")
	sb.WriteString(runtime.GOOS)
	sb.WriteString("\n\n")
	sb.WriteString("Follow the skill instructions above to complete the task.\n")
	sb.WriteString("If you call a tool and get a concrete result, answer directly with that result.\n")
	sb.WriteString("Do not keep retrying the same command unless the previous tool result clearly shows a recoverable error.\n")
	sb.WriteString("For shell_command, prefer returning the weather data to the user instead of describing your internal steps.")
	return sb.String()
}

func displaySkillName(skill *Info) string {
	if skill.Title != "" {
		return skill.Title
	}
	return skill.Name
}

func (r *Runtime) buildUserMessage(task string, ctx map[string]any) string {
	var sb strings.Builder
	sb.WriteString("## Task\n")
	sb.WriteString(task)
	sb.WriteString("\n\n")
	if ctx != nil {
		if input, ok := ctx["user_input"].(string); ok && input != "" {
			sb.WriteString("## User Input\n")
			sb.WriteString(input)
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

func (r *Runtime) convertToolDefinitions(defs []tools.ToolDefinition) []providers.Tool {
	result := make([]providers.Tool, 0, len(defs))
	for _, def := range defs {
		result = append(result, providers.Tool{
			Type: def.Type,
			Function: providers.Function{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters:  def.Function.Parameters,
			},
		})
	}
	return result
}

func (r *Runtime) executeToolCall(ctx context.Context, tc providers.ToolCall) string {
	if r.tools == nil {
		return "error: tool registry is not configured"
	}

	args := map[string]any{}
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return fmt.Sprintf("error: failed to parse tool arguments: %v", err)
		}
	}

	result := r.tools.Execute(ctx, tc.Function.Name, args)
	if result == nil {
		return "error: tool returned nil result"
	}
	if result.Error != nil {
		return fmt.Sprintf("error: %v", result.Error)
	}
	return normalizeToolResult(tc.Function.Name, result.Content)
}

func normalizeToolResult(toolName, content string) string {
	if toolName != "shell_command" || content == "" {
		return content
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return content
	}

	output, _ := payload["output"].(string)
	if output == "" {
		return content
	}

	var parts []string
	parts = append(parts, strings.TrimSpace(output))

	if success, ok := payload["success"].(bool); ok && !success {
		if errMsg, ok := payload["error"].(string); ok && strings.TrimSpace(errMsg) != "" {
			parts = append(parts, "error: "+strings.TrimSpace(errMsg))
		}
		if exitCode, ok := payload["exit_code"]; ok {
			parts = append(parts, fmt.Sprintf("exit_code: %v", exitCode))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func toolCallSignature(calls []providers.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}

	parts := make([]string, 0, len(calls))
	for _, tc := range calls {
		if tc.Function.Name == "" {
			continue
		}
		parts = append(parts, tc.Function.Name+"|"+strings.TrimSpace(tc.Function.Arguments))
	}
	return strings.Join(parts, "\n")
}

func (r *Runtime) ListExecutableSkills(ctx context.Context) ([]*Info, error) {
	return r.loader.List(ctx)
}

func (r *Runtime) GetSkillInfo(ctx context.Context, name string) (*Info, error) {
	return r.loader.LoadInfo(ctx, name)
}

func (r *Runtime) ReadSkillFile(ctx context.Context, name, version string) (*Info, error) {
	path := filepath.Join(".", "skills", fmt.Sprintf("%s-%s", name, version), "SKILL.md")
	parsed, err := r.parser.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return &Info{
		Metadata: Metadata{
			Name:        parsed.Name,
			Description: parsed.Description,
			Version:     parsed.Version,
		},
		Content: parsed.Content,
		Path:    path,
	}, nil
}

type ExecutionPlan struct {
	SkillName string
	Task      string
	Steps     []ExecutionStep
}

type ExecutionStep struct {
	Step    int
	Action  string
	Content string
}

func (r *Runtime) BuildExecutionPlan(ctx context.Context, skillName, task string) (*ExecutionPlan, error) {
	info, err := r.loader.LoadInfo(ctx, skillName)
	if err != nil {
		return nil, fmt.Errorf("failed to load skill: %w", err)
	}

	planPrompt := fmt.Sprintf(`Given the following skill:

# Skill: %s
%s

Task: %s

Break down this task into clear execution steps. Return a numbered list of steps.

Format:
1. [Step description]
2. [Step description]
...
`, displaySkillName(info), info.Content, task)

	messages := []providers.ChatMessage{
		{Role: "user", Content: planPrompt},
	}

	chatResp, err := r.provider.Chat(ctx, providers.ChatRequest{
		Model:    r.modelName,
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build execution plan: %w", err)
	}

	steps := r.parseStepsFromResponse(chatResp.Content)

	return &ExecutionPlan{
		SkillName: skillName,
		Task:      task,
		Steps:     steps,
	}, nil
}

func (r *Runtime) parseStepsFromResponse(response string) []ExecutionStep {
	var steps []ExecutionStep
	lines := strings.Split(response, "\n")
	stepNum := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		for i, ch := range line {
			if ch >= '0' && ch <= '9' {
				continue
			}
			if ch == '.' || ch == ')' || ch == ':' || ch == ' ' || ch == '\t' {
				continue
			}
			if ch == '-' {
				continue
			}
			content := strings.TrimSpace(line[i:])
			if content != "" {
				steps = append(steps, ExecutionStep{
					Step:    stepNum,
					Action:  content,
					Content: content,
				})
				stepNum++
			}
			break
		}
	}
	return steps
}

func (r *Runtime) ValidateSkill(name string) error {
	_, err := r.parser.ParseFile(filepath.Join(".", "skills", name, "SKILL.md"))
	return err
}
