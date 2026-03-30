package tool

import (
	"context"
	"fmt"
	icooclawErrors "icooclaw/pkg/errors"
	skillpkg "icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"icooclaw/pkg/utils"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstallSkillTool allows the LLM agent to install skills from registries.
// It shares the same RegistryManager that FindSkillsTool uses,
// so all registries configured in config are available for installation.
type InstallSkillTool struct {
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

// NewInstallSkillTool creates a new InstallSkillTool.
// registryMgr is the shared registry manager (same instance as FindSkillsTool).
// workspace is the root workspace directory; skills install to {workspace}/skills/{slug}-{version}/.
func NewInstallSkillTool(workspace string, baseURL string, logger *slog.Logger, storage *storage.SkillStorage, timeout time.Duration) *InstallSkillTool {
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

	return &InstallSkillTool{
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

func (t *InstallSkillTool) Name() string {
	return "install_skill"
}

func (t *InstallSkillTool) Description() string {
	return "Install a skill from a registry by slug. Downloads and extracts the skill into the workspace. Use find_skills first to discover available skills."
}

func (t *InstallSkillTool) Parameters() map[string]any {
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

func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
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

	// 安装技能目录
	skillsDir := filepath.Join(t.workspace, "skills")

	// 查找
	installedSkill, err := t.storage.GetSkill(slug)
	if err != nil && err != icooclawErrors.ErrRecordNotFound {
		return tools.ErrorResult(fmt.Sprintf("skill %q not found: %v", slug, err))
	}

	// 如果已经存在，则直接返回
	if installedSkill != nil && installedSkill.Name == slug && (version == "" || installedSkill.Version == version) {
		return tools.SuccessResult(fmt.Sprintf("skill %s-%s already installed", slug, version))
	}

	// 创建安装目录
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to create skills directory: %v", err))
	}

	// 临时文件名
	tmpName := fmt.Sprintf("%s-%s-%d.zip", slug, version, time.Now().Unix())

	// 将文件下载到临时目录
	tmpPath := filepath.Join(t.workspace, "temp", tmpName)

	// Step 3: Download ZIP to temp file (streams in ~32KB chunks).
	u, err := url.Parse(t.baseURL + t.downloadPath)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("invalid base URL: %v", err))
	}

	installVersion := version
	if installVersion == "" {
		installVersion = "latest"
	}
	q := u.Query()
	q.Set("slug", slug)
	if installVersion != "latest" {
		q.Set("version", installVersion)
	}
	u.RawQuery = q.Encode()

	// 下载文件
	tmpPath, err = t.downloadToTempFileWithRetry(ctx, u.String())
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to download file: %v", err))
	}

	extractDir, err := os.MkdirTemp(skillsDir, fmt.Sprintf("%s-*", slug))
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to create temporary extract directory: %v", err))
	}
	defer os.RemoveAll(extractDir)

	// Step 4: Extract from file on disk.
	if err := utils.ExtractZipFile(tmpPath, extractDir); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to extract file: %v", err))
	}

	skillFile, err := findInstalledSkillFile(extractDir)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to locate installed skill file: %v", err))
	}

	parsed, err := skillpkg.NewParser().ParseFile(skillFile)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to parse installed skill: %v", err))
	}

	resolvedVersion := parsed.Version
	if resolvedVersion == "" {
		resolvedVersion = version
	}
	if resolvedVersion == "" {
		return tools.ErrorResult(fmt.Sprintf("failed to determine version for skill %q", slug))
	}

	finalDirName := fmt.Sprintf("%s-%s", parsed.Name, resolvedVersion)
	finalRelativePath := filepath.ToSlash(filepath.Join("skills", finalDirName))
	finalSkillDir := filepath.Join(skillsDir, finalDirName)
	extractedSkillDir := filepath.Dir(skillFile)
	if _, err := os.Stat(finalSkillDir); err == nil {
		return tools.ErrorResult(fmt.Sprintf("skill directory already exists: %s", finalRelativePath))
	} else if !os.IsNotExist(err) {
		return tools.ErrorResult(fmt.Sprintf("failed to check skill directory: %v", err))
	}
	if err := os.Rename(extractedSkillDir, finalSkillDir); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to move skill into place: %v", err))
	}

	if installedSkill == nil {
		installedSkill = &storage.Skill{}
	}
	installedSkill.Name = parsed.Name
	installedSkill.Title = parsed.Title
	installedSkill.Description = parsed.Description
	installedSkill.Path = finalRelativePath
	installedSkill.Version = resolvedVersion
	installedSkill.Enabled = true
	installedSkill.Type = storage.SkillTypeCustom

	// 保存技能配置
	if err := t.storage.SaveSkill(installedSkill); err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to save skill: %v", err))
	}

	return tools.SuccessResult(fmt.Sprintf("skill %s-%s installed", installedSkill.Name, installedSkill.Version))
}

func findInstalledSkillFile(root string) (string, error) {
	var skillFile string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "SKILL.md" {
			skillFile = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && err != fs.SkipAll {
		return "", err
	}
	if skillFile == "" {
		return "", fmt.Errorf("SKILL.md not found under %s", root)
	}
	return skillFile, nil
}

func (t *InstallSkillTool) newGetRequest(ctx context.Context, urlStr, accept string) (*http.Request, error) {
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

func (t *InstallSkillTool) downloadToTempFileWithRetry(ctx context.Context, urlStr string) (string, error) {
	req, err := t.newGetRequest(ctx, urlStr, "application/zip")
	if err != nil {
		return "", err
	}

	tmpPath, err := utils.DownloadToFile(ctx, t.client, req, 0)
	if err != nil {
		return "", err
	}

	if tmpPath == "" {
		return "", fmt.Errorf("download returned empty path")
	}

	return tmpPath, nil
}
