package tool

import (
	"context"
	"fmt"
	"icooclaw/pkg/errors"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type RemoveSkillTool struct {
	workspace string
	mu        sync.Mutex
	logger    *slog.Logger
	storage   *storage.SkillStorage
}

func NewRemoveSkillTool(workspace string, logger *slog.Logger, storage *storage.SkillStorage) *RemoveSkillTool {
	return &RemoveSkillTool{
		workspace: workspace,
		mu:        sync.Mutex{},
		logger:    logger,
		storage:   storage,
	}
}

func (t *RemoveSkillTool) Name() string {
	return "remove_skill"
}

func (t *RemoveSkillTool) Description() string {
	return "Remove a skill from the workspace. Deletes the skill files and removes it from the database. Use find_skills first to see installed skills."
}

func (t *RemoveSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug": map[string]any{
				"type":        "string",
				"description": "The unique slug of the skill to remove (e.g., 'github', 'docker-compose')",
			},
		},
		"required": []string{"slug"},
	}
}

func (t *RemoveSkillTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	t.mu.Lock()
	defer t.mu.Unlock()

	slug, _ := args["slug"].(string)
	if err := utils.ValidateSkillIdentifier(slug); err != nil {
		return tools.ErrorResult(fmt.Sprintf("invalid slug %q: error: %s", slug, err.Error()))
	}

	skill, err := t.storage.GetSkill(slug)
	if err != nil {
		if err == errors.ErrRecordNotFound {
			return tools.ErrorResult(fmt.Sprintf("skill %q is not installed", slug))
		}
		return tools.ErrorResult(fmt.Sprintf("failed to get skill %q: %v", slug, err))
	}

	if skill.Path != "" {
		skillPath := skill.Path
		if !filepath.IsAbs(skillPath) {
			skillPath = filepath.Join(t.workspace, skillPath)
		}
		if err := os.RemoveAll(skillPath); err != nil {
			return tools.ErrorResult(fmt.Sprintf("failed to remove skill directory: %v", err))
		}
		t.logger.Info("Removed skill directory", "path", skillPath)
	}

	if err := t.storage.DeleteSkillByName(slug); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to delete skill from database: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("skill %q removed successfully", slug))
}
