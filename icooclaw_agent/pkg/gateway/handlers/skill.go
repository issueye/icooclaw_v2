package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	icooclawErrors "icooclaw/pkg/errors"
	"icooclaw/pkg/gateway/models"
	skillpkg "icooclaw/pkg/skill"
	skilltool "icooclaw/pkg/skill/tool"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/utils"
)

type SkillHandler struct {
	logger         *slog.Logger
	storage        *storage.Storage
	workspace      string
	installBaseURL string
}

func NewSkillHandler(logger *slog.Logger, storage *storage.Storage) *SkillHandler {
	workspace := ""
	if storage != nil && storage.Workspace() != nil {
		workspace = storage.Workspace().GetWorkspace()
	}
	return &SkillHandler{
		logger:    logger,
		storage:   storage,
		workspace: workspace,
	}
}

type skillTransportItem struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	Content     string `json:"content,omitempty"`
	FileContent string `json:"file_content,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Type        string `json:"type,omitempty"`
}

type skillTransportEnvelope struct {
	Skills []skillTransportItem `json:"skills"`
}

type skillInstallRequest struct {
	Slug    string `json:"slug"`
	Version string `json:"version"`
}

func (h *SkillHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QuerySkill](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	skills, err := h.storage.Skill().Page(req)
	if err != nil {
		h.logger.Error("获取技能列表失败", "error", err)
		http.Error(w, "获取技能列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQuerySkill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

func (h *SkillHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定保存技能请求失败", "error", err)
		http.Error(w, "绑定保存技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
	if err != nil {
		h.logger.Error("保存技能失败", "error", err)
		http.Error(w, "保存技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能保存成功",
		Data:    req,
	})
}

func (h *SkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建技能请求失败", "error", err)
		http.Error(w, "绑定创建技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
	if err != nil {
		h.logger.Error("创建技能失败", "error", err)
		http.Error(w, "创建技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能创建成功",
		Data:    req,
	})
}

func (h *SkillHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定更新技能请求失败", "error", err)
		http.Error(w, "绑定更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
	if err != nil {
		h.logger.Error("更新技能失败", "error", err)
		http.Error(w, "更新技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能更新成功",
		Data:    req,
	})
}

func (h *SkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除技能请求失败", "error", err)
		http.Error(w, "绑定删除技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().DeleteSkill(id)
	if err != nil {
		h.logger.Error("删除技能失败", "error", err)
		http.Error(w, "删除技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "技能删除成功",
	})
}

func (h *SkillHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetSkillByID(id)
	if err != nil {
		h.logger.Error("获取技能失败", "error", err)
		http.Error(w, "获取技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能获取成功",
		Data:    skill,
	})
}

func (h *SkillHandler) GetByName(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[struct {
		Name string `json:"name"`
	}](r)
	if err != nil {
		h.logger.Error("绑定获取技能请求失败", "error", err)
		http.Error(w, "绑定获取技能请求失败", http.StatusBadRequest)
		return
	}

	skill, err := h.storage.Skill().GetSkill(req.Name)
	if err != nil {
		h.logger.Error("获取技能失败", "error", err)
		http.Error(w, "获取技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能获取成功",
		Data:    skill,
	})
}

func (h *SkillHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().ListSkills()
	if err != nil {
		h.logger.Error("获取所有技能失败", "error", err)
		http.Error(w, "获取所有技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能列表获取成功",
		Data:    skills,
	})
}

func (h *SkillHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	skills, err := h.storage.Skill().ListEnabledSkills()
	if err != nil {
		h.logger.Error("获取启用技能失败", "error", err)
		http.Error(w, "获取启用技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]*storage.Skill]{
		Code:    http.StatusOK,
		Message: "启用技能列表获取成功",
		Data:    skills,
	})
}

func (h *SkillHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Skill](r)
	if err != nil {
		h.logger.Error("绑定创建或更新技能请求失败", "error", err)
		http.Error(w, "绑定创建或更新技能请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Skill().SaveSkill(req)
	if err != nil {
		h.logger.Error("创建或更新技能失败", "error", err)
		http.Error(w, "创建或更新技能失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: "技能创建或更新成功",
		Data:    req,
	})
}

func (h *SkillHandler) Export(w http.ResponseWriter, r *http.Request) {
	items, err := h.exportSkillItems()
	if err != nil {
		h.logger.Error("导出技能失败", "error", err)
		http.Error(w, "导出技能失败", http.StatusInternalServerError)
		return
	}

	payload, err := json.MarshalIndent(skillTransportEnvelope{Skills: items}, "", "  ")
	if err != nil {
		h.logger.Error("序列化技能导出内容失败", "error", err)
		http.Error(w, "导出技能失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="skills-export.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (h *SkillHandler) Install(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*skillInstallRequest](r)
	if err != nil {
		h.logger.Error("绑定安装技能请求失败", "error", err)
		http.Error(w, "绑定安装技能请求失败", http.StatusBadRequest)
		return
	}

	if req == nil || strings.TrimSpace(req.Slug) == "" {
		http.Error(w, "slug 不能为空", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(h.workspace) == "" {
		http.Error(w, "工作空间未初始化", http.StatusInternalServerError)
		return
	}

	tool := skilltool.NewInstallSkillTool(
		h.workspace,
		h.installBaseURL,
		h.logger,
		h.storage.Skill(),
		30*time.Second,
	)
	result := tool.Execute(r.Context(), map[string]any{
		"slug":    req.Slug,
		"version": req.Version,
	})
	if result == nil {
		http.Error(w, "安装技能失败", http.StatusInternalServerError)
		return
	}
	if !result.Success {
		http.Error(w, result.Content, http.StatusBadRequest)
		return
	}

	skillInfo, err := h.storage.Skill().GetSkill(req.Slug)
	if err != nil {
		h.logger.Warn("安装技能后读取技能记录失败", "slug", req.Slug, "error", err)
		models.WriteData(w, models.BaseResponse[map[string]string]{
			Code:    http.StatusOK,
			Message: result.Content,
			Data:    map[string]string{"slug": req.Slug},
		})
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Skill]{
		Code:    http.StatusOK,
		Message: result.Content,
		Data:    skillInfo,
	})
}

// Import 导入技能
func (h *SkillHandler) Import(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		h.importZip(w, r)
		return
	}

	req, err := models.Bind[struct {
		Data      string `json:"data"`
		Overwrite bool   `json:"overwrite"`
	}](r)
	if err != nil {
		h.logger.Error("绑定导入技能请求失败", "error", err)
		http.Error(w, "绑定导入技能请求失败", http.StatusBadRequest)
		return
	}

	items, err := decodeSkillImportItems(req.Data)
	if err != nil {
		h.logger.Error("解析导入技能内容失败", "error", err)
		http.Error(w, "解析导入技能内容失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	success, skip, err := h.importSkillItems(r.Context(), items, req.Overwrite)
	if err != nil {
		h.logger.Error("导入技能失败", "error", err)
		http.Error(w, "导入技能失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]int]{
		Code:    http.StatusOK,
		Message: "技能导入成功",
		Data:    map[string]int{"success": success, "skip": skip},
	})
}

func (h *SkillHandler) importZip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.logger.Error("解析导入技能表单失败", "error", err)
		http.Error(w, "解析导入技能表单失败", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.logger.Error("读取导入技能文件失败", "error", err)
		http.Error(w, "读取导入技能文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	overwrite := strings.EqualFold(strings.TrimSpace(r.FormValue("overwrite")), "true")
	tmpFile, err := os.CreateTemp("", "skill-import-*.zip")
	if err != nil {
		h.logger.Error("创建临时导入文件失败", "error", err)
		http.Error(w, "创建临时导入文件失败", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.ReadFrom(file); err != nil {
		_ = tmpFile.Close()
		h.logger.Error("保存上传的技能压缩包失败", "error", err)
		http.Error(w, "保存上传的技能压缩包失败", http.StatusBadRequest)
		return
	}
	if err := tmpFile.Close(); err != nil {
		h.logger.Error("关闭临时导入文件失败", "error", err)
		http.Error(w, "关闭临时导入文件失败", http.StatusInternalServerError)
		return
	}

	extractDir, err := os.MkdirTemp("", "skill-import-*")
	if err != nil {
		h.logger.Error("创建导入解压目录失败", "error", err)
		http.Error(w, "创建导入解压目录失败", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(extractDir)

	if err := utils.ExtractZipFile(tmpPath, extractDir); err != nil {
		h.logger.Error("解压技能压缩包失败", "filename", header.Filename, "error", err)
		http.Error(w, "解压技能压缩包失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	items, err := collectSkillItemsFromDir(extractDir)
	if err != nil {
		h.logger.Error("扫描导入技能失败", "error", err)
		http.Error(w, "扫描导入技能失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	success, skip, err := h.importSkillItems(r.Context(), items, overwrite)
	if err != nil {
		h.logger.Error("导入技能失败", "error", err)
		http.Error(w, "导入技能失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[map[string]int]{
		Code:    http.StatusOK,
		Message: "技能导入成功",
		Data:    map[string]int{"success": success, "skip": skip},
	})
}

func decodeSkillImportItems(raw string) ([]skillTransportItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("导入内容不能为空")
	}

	var envelope skillTransportEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && len(envelope.Skills) > 0 {
		return envelope.Skills, nil
	}

	var items []skillTransportItem
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return items, nil
	}

	var single skillTransportItem
	if err := json.Unmarshal([]byte(raw), &single); err == nil && strings.TrimSpace(single.Name) != "" {
		return []skillTransportItem{single}, nil
	}

	return nil, fmt.Errorf("不支持的导入格式")
}

func (h *SkillHandler) exportSkillItems() ([]skillTransportItem, error) {
	if h.storage == nil {
		return nil, fmt.Errorf("storage 未初始化")
	}

	skills, err := h.storage.Skill().ListSkills()
	if err != nil {
		return nil, err
	}

	items := make([]skillTransportItem, 0, len(skills))
	for _, sk := range skills {
		if sk == nil || sk.Type == storage.SkillTypeSkill {
			continue
		}
		skillFile, err := h.resolveSkillFilePath(sk)
		if err != nil {
			h.logger.Warn("跳过无法解析路径的技能", "name", sk.Name, "error", err)
			continue
		}
		content, err := os.ReadFile(skillFile)
		if err != nil {
			h.logger.Warn("跳过无法读取文件的技能", "name", sk.Name, "path", skillFile, "error", err)
			continue
		}
		enabled := sk.Enabled
		items = append(items, skillTransportItem{
			Name:        sk.Name,
			Title:       sk.Title,
			Description: sk.Description,
			Version:     sk.Version,
			FileContent: string(content),
			Enabled:     &enabled,
			Type:        sk.Type.String(),
		})
	}

	return items, nil
}

func (h *SkillHandler) importSkillItems(ctx context.Context, items []skillTransportItem, overwrite bool) (int, int, error) {
	if h.storage == nil || h.storage.Skill() == nil {
		return 0, 0, fmt.Errorf("storage 未初始化")
	}
	if strings.TrimSpace(h.workspace) == "" {
		return 0, 0, fmt.Errorf("工作空间未初始化")
	}

	success := 0
	skip := 0
	parser := skillpkg.NewParser()
	for _, item := range items {
		identifier := firstNonEmpty(item.Name, item.Title)
		existing, err := h.storage.Skill().GetSkill(identifier)
		if err != nil && err != icooclawErrors.ErrRecordNotFound {
			return success, skip, err
		}
		if existing != nil && existing.Type == storage.SkillTypeSkill {
			skip++
			continue
		}
		if existing != nil && !overwrite {
			skip++
			continue
		}

		fileContent := strings.TrimSpace(item.FileContent)
		if fileContent == "" {
			fileContent, err = buildImportedSkillFileContent(item)
			if err != nil {
				return success, skip, err
			}
		}

		parsed, err := parser.Parse(fileContent)
		if err != nil {
			return success, skip, fmt.Errorf("解析技能 %s 失败: %w", identifier, err)
		}

		resolvedVersion := firstNonEmpty(parsed.Version, item.Version)
		finalDirName := parsed.Name
		if strings.TrimSpace(resolvedVersion) != "" {
			finalDirName = fmt.Sprintf("%s-%s", parsed.Name, resolvedVersion)
		}
		finalRelativePath := filepath.ToSlash(filepath.Join("skills", finalDirName))
		finalSkillDir := filepath.Join(h.workspace, filepath.FromSlash(finalRelativePath))

		if existing != nil {
			if err := h.removeExistingSkillDir(existing); err != nil {
				return success, skip, err
			}
		}
		if err := os.MkdirAll(finalSkillDir, 0o755); err != nil {
			return success, skip, fmt.Errorf("创建技能目录失败: %w", err)
		}
		if err := utils.WriteFileAtomic(filepath.Join(finalSkillDir, "SKILL.md"), []byte(fileContent), 0o644); err != nil {
			return success, skip, fmt.Errorf("写入技能文件失败: %w", err)
		}

		skillRecord := &storage.Skill{
			Name:        parsed.Name,
			Title:       firstNonEmpty(item.Title, parsed.Title, parsed.Name),
			Description: parsed.Description,
			Version:     resolvedVersion,
			Path:        finalRelativePath,
			Type:        storage.SkillTypeCustom,
			Enabled:     true,
		}
		if existing != nil {
			skillRecord.ID = existing.ID
			skillRecord.Enabled = existing.Enabled
		}
		if item.Enabled != nil {
			skillRecord.Enabled = *item.Enabled
		}

		if err := h.storage.Skill().SaveSkill(skillRecord); err != nil {
			return success, skip, fmt.Errorf("保存技能记录失败: %w", err)
		}
		success++
	}

	_ = ctx
	return success, skip, nil
}

func buildImportedSkillFileContent(item skillTransportItem) (string, error) {
	content := strings.TrimSpace(item.Content)
	if content == "" {
		return "", fmt.Errorf("技能 %s 缺少内容", firstNonEmpty(item.Name, item.Title))
	}

	skill := &skillpkg.ParsedSkill{
		Name:        strings.TrimSpace(item.Name),
		Title:       strings.TrimSpace(item.Title),
		Version:     strings.TrimSpace(item.Version),
		Description: strings.TrimSpace(item.Description),
		Content:     content,
	}
	if skill.Name == "" {
		skill.Name = utils.NormalizeSkillIdentifier(item.Title)
	}
	if skill.Title == "" {
		skill.Title = skill.Name
	}

	parser := skillpkg.NewParser()
	if err := parser.Validate(skill); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	displayName := firstNonEmpty(skill.Title, skill.Name)
	sb.WriteString(fmt.Sprintf("name: %s\n", displayName))
	if skill.Name != "" && skill.Name != displayName {
		sb.WriteString(fmt.Sprintf("slug: %s\n", skill.Name))
	}
	if skill.Version != "" {
		sb.WriteString(fmt.Sprintf("version: %s\n", skill.Version))
	}
	sb.WriteString(fmt.Sprintf("description: %s\n", skill.Description))
	sb.WriteString("---\n\n")
	sb.WriteString(skill.Content)
	return sb.String(), nil
}

func collectSkillItemsFromDir(root string) ([]skillTransportItem, error) {
	items := make([]skillTransportItem, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "SKILL.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parsed, err := skillpkg.NewParser().Parse(string(content))
		if err != nil {
			return fmt.Errorf("解析 %s 失败: %w", path, err)
		}
		items = append(items, skillTransportItem{
			Name:        parsed.Name,
			Title:       firstNonEmpty(parsed.Title, parsed.Name),
			Description: parsed.Description,
			Version:     parsed.Version,
			FileContent: string(content),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("zip 中未找到任何 SKILL.md")
	}
	return items, nil
}

func (h *SkillHandler) removeExistingSkillDir(skill *storage.Skill) error {
	if skill == nil || strings.TrimSpace(skill.Path) == "" {
		return nil
	}

	resolved, err := h.resolvePathUnderWorkspace(skill.Path)
	if err != nil {
		return nil
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(resolved)
	}
	return os.RemoveAll(filepath.Dir(resolved))
}

func (h *SkillHandler) resolveSkillFilePath(skill *storage.Skill) (string, error) {
	if skill == nil {
		return "", fmt.Errorf("skill 不能为空")
	}
	resolved, err := h.resolvePathUnderWorkspace(skill.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Join(resolved, "SKILL.md"), nil
	}
	return resolved, nil
}

func (h *SkillHandler) resolvePathUnderWorkspace(relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	if strings.TrimSpace(h.workspace) == "" {
		return "", fmt.Errorf("工作空间未初始化")
	}

	base := filepath.Clean(h.workspace)
	target := relativePath
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, filepath.FromSlash(relativePath))
	}
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("非法技能路径: %s", relativePath)
	}
	return target, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
