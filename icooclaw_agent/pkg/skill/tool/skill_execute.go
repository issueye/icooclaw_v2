package tool

import (
	"context"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"log/slog"
	"slices"
	"strings"
)

type ExecuteSkillTool struct {
	workspace       string
	storage         *storage.Storage
	providerManager *providers.Manager
	logger          *slog.Logger
	tools           *tools.Registry
}

func NewExecuteSkillTool(
	workspace string,
	storage *storage.Storage,
	providerManager *providers.Manager,
	toolRegistry *tools.Registry,
	logger *slog.Logger,
) *ExecuteSkillTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExecuteSkillTool{
		workspace:       workspace,
		storage:         storage,
		providerManager: providerManager,
		tools:           toolRegistry,
		logger:          logger,
	}
}

func (t *ExecuteSkillTool) Name() string {
	return "skill_execute"
}

func (t *ExecuteSkillTool) Description() string {
	return "Execute an installed skill to perform a specific task. The skill provides specialized instructions and workflows for the LLM to follow. Use this when a user asks for something that matches a skill's purpose."
}

func (t *ExecuteSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"skill_name": map[string]any{
				"type":        "string",
				"description": "The name of the skill to execute (e.g., 'skill-creator', 'github')",
			},
			"task": map[string]any{
				"type":        "string",
				"description": "The task to perform using the skill's instructions",
			},
			"context": map[string]any{
				"type":        "object",
				"description": "Additional context for the skill execution (optional)",
				"properties": map[string]any{
					"skill_notes": map[string]any{
						"type":        "string",
						"description": "Additional notes or context for the skill",
					},
					"user_input": map[string]any{
						"type":        "string",
						"description": "User-provided input data for the skill",
					},
				},
			},
		},
		"required": []string{"skill_name", "task"},
	}
}

func (t *ExecuteSkillTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return tools.ErrorResult("skill_name is required")
	}

	task, ok := args["task"].(string)
	if !ok || task == "" {
		return tools.ErrorResult("task is required")
	}
	if t.storage == nil {
		return tools.ErrorResult("skill storage is not configured")
	}

	provider, modelName, err := t.resolveProvider()
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("skill provider is not configured: %v", err))
	}

	var skillCtx map[string]any
	if ctxArg, ok := args["context"].(map[string]any); ok {
		skillCtx = ctxArg
	}

	skillLoader := skill.NewLoader(t.workspace, t.storage, t.logger)

	runtime := skill.NewRuntime(skill.RuntimeConfig{
		Loader:    skillLoader,
		Provider:  provider,
		ModelName: modelName,
		Tools:     skillTools(t.tools),
		Logger:    t.logger,
	})

	execReq := skill.ExecutionRequest{
		SkillName: skillName,
		Task:      task,
		Context:   skillCtx,
	}

	result := runtime.Execute(ctx, execReq)
	if result.Error != nil {
		return tools.ErrorResult(fmt.Sprintf("skill execution failed [%s]: %v", formatSkillName(result.SkillName, result.DisplayName, skillName), result.Error))
	}

	return tools.SuccessResult(formatSkillExecutionOutput(formatSkillName(result.SkillName, result.DisplayName, skillName), result.Output))
}

func (t *ExecuteSkillTool) resolveProvider() (providers.Provider, string, error) {
	if t.storage == nil {
		return nil, "", fmt.Errorf("skill storage is not configured")
	}
	if t.providerManager == nil {
		return nil, "", fmt.Errorf("provider manager is not configured")
	}

	if modelRef, ok := t.loadModelParam(consts.SKILL_DEFAULT_MODEL_KEY); ok {
		return t.resolveConfiguredProvider(modelRef, true)
	}

	if modelRef, ok := t.loadModelParam(consts.DEFAULT_MODEL_KEY); ok {
		provider, modelName, err := t.resolveConfiguredProvider(modelRef, false)
		if err == nil {
			return provider, modelName, nil
		}
		t.logger.Warn("默认 Agent 模型不适合 skill_execute，尝试回退到普通聊天提供商", "model", modelRef, "error", err)
	}

	return t.resolveFallbackProvider()
}

func (t *ExecuteSkillTool) loadModelParam(key string) (string, bool) {
	if t.storage == nil {
		return "", false
	}

	param, err := t.storage.Param().Get(key)
	if err != nil || param == nil {
		return "", false
	}

	value := strings.TrimSpace(param.Value)
	if value == "" {
		return "", false
	}
	return value, true
}

func (t *ExecuteSkillTool) resolveConfiguredProvider(modelRef string, explicit bool) (providers.Provider, string, error) {
	parts := utils.SplitProviderModel(modelRef)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid provider/model format: %s", modelRef)
	}

	providerName := strings.TrimSpace(parts[0])
	modelName := strings.TrimSpace(parts[1])
	if providerName == "" || modelName == "" {
		return nil, "", fmt.Errorf("invalid provider/model format: %s", modelRef)
	}

	cfg, err := t.storage.Provider().GetByName(providerName)
	if err != nil {
		return nil, "", fmt.Errorf("provider %s not found: %w", providerName, err)
	}
	if !cfg.Enabled {
		return nil, "", fmt.Errorf("provider %s is disabled", providerName)
	}
	resolvedModelName := resolveSkillProviderModelName(cfg, modelName)
	if err := validateSkillProviderCandidate(cfg, resolvedModelName); err != nil {
		if explicit {
			return nil, "", fmt.Errorf("%w; please configure %s to a supported model", err, consts.SKILL_DEFAULT_MODEL_KEY)
		}
		return nil, "", err
	}

	provider, err := t.providerManager.Get(providerName)
	if err != nil {
		return nil, "", err
	}
	return provider, resolvedModelName, nil
}

func (t *ExecuteSkillTool) resolveFallbackProvider() (providers.Provider, string, error) {
	configs, err := t.storage.Provider().List()
	if err != nil {
		return nil, "", fmt.Errorf("list providers failed: %w", err)
	}

	for _, cfg := range configs {
		if cfg == nil || !cfg.Enabled {
			continue
		}
		provider, err := t.providerManager.Get(cfg.Name)
		if err != nil {
			continue
		}

		for _, modelName := range skillProviderCandidateModels(cfg) {
			if err := validateSkillProviderCandidate(cfg, modelName); err != nil {
				continue
			}

			t.logger.Info("skill_execute fallback provider selected", "provider", cfg.Name, "model", modelName)
			return provider, modelName, nil
		}
	}

	return nil, "", fmt.Errorf("no compatible provider available for skill execution; configure %s", consts.SKILL_DEFAULT_MODEL_KEY)
}

func validateSkillProviderCandidate(cfg *storage.Provider, modelName string) error {
	if cfg == nil {
		return fmt.Errorf("provider config is nil")
	}

	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return fmt.Errorf("provider %s does not define a usable model", cfg.Name)
	}

	if !isSkillCodingProvider(cfg) {
		return nil
	}

	if !isSupportedSkillCodingModel(modelName) {
		return fmt.Errorf("provider %s uses coding endpoint %s and model %s is not supported for skill execution", cfg.Name, cfg.APIBase, modelName)
	}
	return nil
}

func isSkillCodingProvider(cfg *storage.Provider) bool {
	if cfg == nil {
		return false
	}
	if cfg.Type == consts.ProviderQwenCodingPlan {
		return true
	}
	apiBase := strings.ToLower(strings.TrimSpace(cfg.APIBase))
	return strings.Contains(apiBase, "/coding")
}

func isSupportedSkillCodingModel(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "qianfan-code-latest", "kimi-k2.5", "glm-5", "deepseek-v3.2", "minimax-m2.5":
		return true
	default:
		return false
	}
}

func resolveSkillProviderModelName(cfg *storage.Provider, modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if cfg == nil || modelName == "" {
		return modelName
	}

	for _, llm := range cfg.LLMs {
		alias := strings.TrimSpace(llm.Alias)
		model := strings.TrimSpace(llm.Model)
		if alias != "" && strings.EqualFold(alias, modelName) && model != "" {
			return model
		}
		if model != "" && strings.EqualFold(model, modelName) {
			return model
		}
	}

	return modelName
}

func skillProviderCandidateModels(cfg *storage.Provider) []string {
	if cfg == nil {
		return nil
	}

	models := make([]string, 0, len(cfg.LLMs)+1)
	addModel := func(modelName string) {
		resolved := resolveSkillProviderModelName(cfg, modelName)
		if resolved == "" || slices.Contains(models, resolved) {
			return
		}
		models = append(models, resolved)
	}

	addModel(cfg.DefaultModel)
	for _, llm := range cfg.LLMs {
		addModel(llm.Model)
		addModel(llm.Alias)
	}

	return models
}

func skillTools(registry *tools.Registry) *tools.Registry {
	if registry == nil {
		return nil
	}
	return registry.CloneWithout("skill_execute")
}

func formatSkillName(skillName, displayName, fallback string) string {
	if displayName != "" {
		if skillName != "" && skillName != displayName {
			return fmt.Sprintf("%s [%s]", displayName, skillName)
		}
		return displayName
	}
	if skillName != "" {
		return skillName
	}
	return fallback
}

func formatSkillExecutionOutput(skillLabel, output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return fmt.Sprintf("技能: %s", skillLabel)
	}
	return fmt.Sprintf("技能: %s\n\n%s", skillLabel, output)
}

type ListSkillsTool struct {
	storage *storage.Storage
	logger  *slog.Logger
}

func NewListSkillsTool(storage *storage.Storage, logger *slog.Logger) *ListSkillsTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &ListSkillsTool{
		storage: storage,
		logger:  logger,
	}
}

func (t *ListSkillsTool) Name() string {
	return "skill_list"
}

func (t *ListSkillsTool) Description() string {
	return "List all installed skills available for execution. Use this to discover what skills are currently installed and ready to use."
}

func (t *ListSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *ListSkillsTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	if t.storage == nil {
		return tools.SuccessResult("No skills installed. Use install_skill to install skills from the registry.")
	}

	skills, err := t.storage.Skill().ListEnabledSkills()
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to list skills: %v", err))
	}

	if len(skills) == 0 {
		return tools.SuccessResult("No skills installed. Use install_skill to install skills from the registry.")
	}

	var output string
	for _, s := range skills {
		displayName := s.Title
		if displayName == "" {
			displayName = s.Name
		}
		output += fmt.Sprintf("- **%s** [%s] (v%s): %s\n", displayName, s.Name, s.Version, s.Description)
	}

	return tools.SuccessResult(output)
}

type GetSkillInfoTool struct {
	storage *storage.Storage
	logger  *slog.Logger
}

func NewGetSkillInfoTool(storage *storage.Storage, logger *slog.Logger) *GetSkillInfoTool {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetSkillInfoTool{
		storage: storage,
		logger:  logger,
	}
}

func (t *GetSkillInfoTool) Name() string {
	return "skill_info"
}

func (t *GetSkillInfoTool) Description() string {
	return "Get detailed information about a specific installed skill, including its full description and instructions."
}

func (t *GetSkillInfoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"skill_name": map[string]any{
				"type":        "string",
				"description": "The name of the skill to get information about",
			},
		},
		"required": []string{"skill_name"},
	}
}

func (t *GetSkillInfoTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		return tools.ErrorResult("skill_name is required")
	}
	if t.storage == nil {
		return tools.ErrorResult("skill storage is not configured")
	}

	s, err := t.storage.Skill().GetSkill(skillName)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("skill %q not found: %v", skillName, err))
	}

	displayName := s.Title
	if displayName == "" {
		displayName = s.Name
	}
	info := fmt.Sprintf("## %s [%s] (v%s)\n\n%s\n\n**Path:** %s", displayName, s.Name, s.Version, s.Description, s.Path)
	return tools.SuccessResult(info)
}
