package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/gateway/models"
	"icooclaw/pkg/scheduler"
	"icooclaw/pkg/storage"
)

type TaskHandler struct {
	logger   *slog.Logger
	storage  *storage.Storage
	schedule *scheduler.Scheduler
}

func NewTaskHandler(logger *slog.Logger, storage *storage.Storage, schedule *scheduler.Scheduler) *TaskHandler {
	return &TaskHandler{logger: logger, storage: storage, schedule: schedule}
}

func normalizeTaskChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "":
		return ""
	case consts.FEISHU, strings.ToLower(consts.FEISHU_CN):
		return consts.FEISHU
	case consts.DINGTALK, "钉钉":
		return consts.DINGTALK
	case consts.WEBSOCKET:
		return consts.WEBSOCKET
	case consts.ICOO_CHAT:
		return consts.ICOO_CHAT
	case consts.QQ:
		return consts.QQ
	case consts.TELEGRAM:
		return consts.TELEGRAM
	default:
		return strings.TrimSpace(channel)
	}
}

func (h *TaskHandler) normalizeTaskForSave(task *storage.Task) error {
	if task == nil {
		return nil
	}

	if task.TaskType == "" {
		task.TaskType = scheduler.TaskTypeScheduled
	}
	if task.TaskType != scheduler.TaskTypeImmediate && task.TaskType != scheduler.TaskTypeScheduled {
		return fmt.Errorf("不支持的任务类型: %s", task.TaskType)
	}
	if task.Executor == "" {
		task.Executor = scheduler.TaskExecutorMessage
	}
	task.Channel = normalizeTaskChannel(task.Channel)

	if task.TaskType == scheduler.TaskTypeScheduled {
		normalized, err := scheduler.NormalizeSchedule(task.CronExpr)
		if err != nil {
			return err
		}
		task.CronExpr = normalized
		return nil
	}

	task.CronExpr = ""
	return nil
}

func (h *TaskHandler) syncTask(task *storage.Task) error {
	if h.schedule == nil || task == nil {
		return nil
	}
	_, err := h.schedule.ApplyStorageTask(task)
	return err
}

func (h *TaskHandler) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.QueryTask](r)
	if err != nil {
		h.logger.Error("绑定分页请求失败", "error", err)
		http.Error(w, "绑定分页请求失败", http.StatusBadRequest)
		return
	}

	tasks, err := h.storage.Task().Page(req)
	if err != nil {
		h.logger.Error("获取任务列表失败", "error", err)
		http.Error(w, "获取任务列表失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.ResQueryTask]{
		Code:    http.StatusOK,
		Message: "任务列表获取成功",
		Data:    tasks,
	})
}

// Save doc 保存任务
func (h *TaskHandler) Save(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定保存任务请求失败", "error", err)
		http.Error(w, "绑定保存任务请求失败", http.StatusBadRequest)
		return
	}

	if err := h.normalizeTaskForSave(req); err != nil {
		h.logger.Error("任务调度表达式无效", "error", err)
		http.Error(w, "任务调度表达式无效: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.storage.Task().CreateOrUpdate(req)
	if err != nil {
		h.logger.Error("保存任务失败", "error", err)
		http.Error(w, "保存任务失败", http.StatusInternalServerError)
		return
	}

	if err := h.syncTask(req); err != nil {
		h.logger.Error("同步调度器任务失败", "id", req.ID, "error", err)
		http.Error(w, "同步调度器任务失败", http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务保存成功",
		Data:    req,
	})
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定创建任务请求失败", "error", err)
		http.Error(w, "绑定创建任务请求失败", http.StatusBadRequest)
		return
	}

	if err := h.normalizeTaskForSave(req); err != nil {
		h.logger.Error("任务调度表达式无效", "error", err)
		http.Error(w, "任务调度表达式无效: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Create(req)
	if err != nil {
		h.logger.Error("创建任务失败", "error", err)
		http.Error(w, "创建任务失败", http.StatusInternalServerError)
		return
	}

	if err := h.syncTask(req); err != nil {
		h.logger.Error("同步调度器任务失败", "id", req.ID, "error", err)
		http.Error(w, "同步调度器任务失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务创建成功",
		Data:    req,
	})
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[*storage.Task](r)
	if err != nil {
		h.logger.Error("绑定更新任务请求失败", "error", err)
		http.Error(w, "绑定更新任务请求失败", http.StatusBadRequest)
		return
	}

	if err := h.normalizeTaskForSave(req); err != nil {
		h.logger.Error("任务调度表达式无效", "error", err)
		http.Error(w, "任务调度表达式无效: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Update(req)
	if err != nil {
		h.logger.Error("更新任务失败", "error", err)
		http.Error(w, "更新任务失败", http.StatusInternalServerError)
		return
	}

	if err := h.syncTask(req); err != nil {
		h.logger.Error("同步调度器任务失败", "id", req.ID, "error", err)
		http.Error(w, "同步调度器任务失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务更新成功",
		Data:    req,
	})
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定删除任务请求失败", "error", err)
		http.Error(w, "绑定删除任务请求失败", http.StatusBadRequest)
		return
	}

	err = h.storage.Task().Delete(id)
	if err != nil {
		h.logger.Error("删除任务失败", "error", err)
		http.Error(w, "删除任务失败", http.StatusInternalServerError)
		return
	}

	if h.schedule != nil {
		if err := h.schedule.DeleteTask(id); err != nil {
			h.logger.Warn("删除调度器任务失败", "id", id, "error", err)
		}
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务删除成功",
	})
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定获取任务请求失败", "error", err)
		http.Error(w, "绑定获取任务请求失败", http.StatusBadRequest)
		return
	}

	task, err := h.storage.Task().GetByID(id)
	if err != nil {
		h.logger.Error("获取任务失败", "error", err)
		http.Error(w, "获取任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[*storage.Task]{
		Code:    http.StatusOK,
		Message: "任务获取成功",
		Data:    task,
	})
}

// ToggleEnabled 切换任务状态
func (h *TaskHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定切换任务状态请求失败", "error", err)
		http.Error(w, "绑定切换任务状态请求失败", http.StatusBadRequest)
		return
	}

	// 切换任务状态
	task, err := h.storage.Task().ToggleEnabled(id)
	if err != nil {
		h.logger.Error("切换任务状态失败", "error", err)
		http.Error(w, "切换任务状态失败", http.StatusInternalServerError)
		return
	}

	// 同步调度器状态
	if err := h.syncTask(task); err != nil {
		h.logger.Error("同步调度器任务失败", "id", task.ID, "error", err)
		http.Error(w, "同步调度器任务失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 返回响应
	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务状态切换成功",
	})
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.storage.Task().GetAll()
	if err != nil {
		h.logger.Error("获取所有任务失败", "error", err)
		http.Error(w, "获取所有任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Task]{
		Code:    http.StatusOK,
		Message: "任务列表获取成功",
		Data:    tasks,
	})
}

func (h *TaskHandler) GetEnabled(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.storage.Task().GetEnabled()
	if err != nil {
		h.logger.Error("获取启用任务失败", "error", err)
		http.Error(w, "获取启用任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]storage.Task]{
		Code:    http.StatusOK,
		Message: "启用任务列表获取成功",
		Data:    tasks,
	})
}

// Execute 立即执行任务
func (h *TaskHandler) Execute(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error("绑定立即执行任务请求失败", "error", err)
		http.Error(w, "绑定立即执行任务请求失败", http.StatusBadRequest)
		return
	}

	// 立即执行任务
	_, err = h.schedule.RunTask(id)
	if err != nil {
		h.logger.Error("立即执行任务失败", "error", err)
		http.Error(w, "立即执行任务失败", http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: "任务执行指令已发送",
	})
}
