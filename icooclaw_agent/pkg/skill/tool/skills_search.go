package tool

import (
	"context"
	"encoding/json"
	"fmt"
	icooclawErrors "icooclaw/pkg/errors"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type SkillSearchResponse struct {
	Skill         SkillInfo     `json:"skill"`
	LatestVersion LatestVersion `json:"latestVersion"`
	Metadata      SkillMetadata `json:"metadata"`
	Owner         OwnerInfo     `json:"owner"`
	Moderation    interface{}   `json:"moderation"`
}

type SkillInfo struct {
	Slug        string         `json:"slug"`
	DisplayName string         `json:"displayName"`
	Summary     string         `json:"summary"`
	Tags        map[string]any `json:"tags"`
	Stats       SkillStats     `json:"stats"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

type SkillStats struct {
	Comments        int `json:"comments"`
	Downloads       int `json:"downloads"`
	InstallsAllTime int `json:"installsAllTime"`
	InstallsCurrent int `json:"installsCurrent"`
	Stars           int `json:"stars"`
	Versions        int `json:"versions"`
}

type LatestVersion struct {
	Version   string `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	Changelog string `json:"changelog"`
	License   string `json:"license"`
}

type SkillMetadata struct {
	OS      string `json:"os"`
	Systems string `json:"systems"`
}

type OwnerInfo struct {
	Handle      string `json:"handle"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Image       string `json:"image"`
}

// SearchSkillTool allows the LLM agent to install skills from registries.
// It shares the same RegistryManager that FindSkillsTool uses,
// so all registries configured in config are available for installation.
type SearchSkillTool struct {
	workspace    string
	mu           sync.Mutex
	logger       *slog.Logger
	storage      *storage.SkillStorage
	authToken    string
	client       *http.Client
	baseURL      string
	searchPath   string
	skillsPath   string
	downloadPath string
}

// NewInstallSkillTool creates a new SearchSkillTool.
// registryMgr is the shared registry manager (same instance as FindSkillsTool).
// workspace is the root workspace directory; skills install to {workspace}/skills/{slug}/.
func NewSearchSkillTool(workspace string, baseURL string, logger *slog.Logger, storage *storage.SkillStorage, timeout time.Duration) *SearchSkillTool {
	if baseURL == "" {
		baseURL = "https://clawhub.ai"
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	return &SearchSkillTool{
		workspace:    workspace,
		mu:           sync.Mutex{},
		logger:       logger,
		storage:      storage,
		client:       client,
		baseURL:      baseURL,
		searchPath:   "/api/v1/search",
		skillsPath:   "/api/v1/skills",
		downloadPath: "/api/v1/download",
	}
}

func (t *SearchSkillTool) Name() string {
	return "find_skills"
}

func (t *SearchSkillTool) Description() string {
	return "Search for installable skills from skill registries. Returns skill slugs, descriptions, versions, and relevance scores. Use this to discover skills before installing them with install_skill."
}

func (t *SearchSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"slug": map[string]any{
				"type":        "string",
				"description": "The unique slug of the skill to install (e.g., 'github', 'docker-compose')",
			},
			"version": map[string]any{
				"type":        "string",
				"description": "Specific version to install (optional, defaults to latest)",
			},
		},
		"required": []string{"slug"},
	}
}

func (t *SearchSkillTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	// Install lock to prevent concurrent directory operations.
	// Ideally this should be done at a `slug` level, currently, its at a `workspace` level.
	t.mu.Lock()
	defer t.mu.Unlock()

	// Validate 验证技能标识符 slug
	slug, _ := args["slug"].(string)
	if err := utils.ValidateSkillIdentifier(slug); err != nil {
		return tools.ErrorResult(fmt.Sprintf("invalid slug %q: error: %s", slug, err.Error()))
	}

	version, _ := args["version"].(string)

	// 查找
	skill, err := t.storage.GetSkill(slug)
	if err != nil && err != icooclawErrors.ErrRecordNotFound {
		return tools.ErrorResult(fmt.Sprintf("skill %q not found: %v", slug, err))
	}

	// 如果已经存在，则直接返回
	if skill != nil && skill.Name == slug && (version == "" || skill.Version == version) {
		return tools.SuccessResult(fmt.Sprintf("skill %s-%s already installed", slug, version))
	}

	// Step 3: Search for skill info from registry API
	u, err := url.Parse(t.baseURL + t.searchPath + "?q=" + slug)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("invalid base URL: %v", err))
	}

	req, err := t.newGetRequest(ctx, u.String(), "application/json")
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	resp, err := utils.DoRequestWithRetry(t.client, req)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to search skill: %v", err))
	}
	defer resp.Body.Close()

	// 解析响应体，获取技能信息
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	var skillResp SkillSearchResponse
	if err := json.Unmarshal(body, &skillResp); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to parse skill info: %v", err))
	}

	// 构建技能信息输出
	info := fmt.Sprintf("Skill: %s (%s)\n", skillResp.Skill.DisplayName, skillResp.Skill.Slug)
	info += fmt.Sprintf("Summary: %s\n", skillResp.Skill.Summary)
	info += fmt.Sprintf("Version: %s\n", skillResp.LatestVersion.Version)
	info += fmt.Sprintf("Author: %s (%s)\n", skillResp.Owner.DisplayName, skillResp.Owner.Handle)
	info += fmt.Sprintf("Stats: %d stars, %d downloads\n", skillResp.Skill.Stats.Stars, skillResp.Skill.Stats.Downloads)
	info += fmt.Sprintf("Changelog:\n%s", skillResp.LatestVersion.Changelog)

	return tools.SuccessResult(info)
}

func (t *SearchSkillTool) newGetRequest(ctx context.Context, urlStr, accept string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	if t.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.authToken)
	}
	return req, nil
}

func (t *SearchSkillTool) downloadToTempFileWithRetry(ctx context.Context, urlStr string) (string, error) {
	req, err := t.newGetRequest(ctx, urlStr, "application/zip")
	if err != nil {
		return "", err
	}

	resp, err := utils.DoRequestWithRetry(t.client, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, errBody)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody[:n]))
	}

	return "", nil
}
