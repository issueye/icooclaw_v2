package handlers

import (
	"log/slog"
	"net/http"

	"icooclaw/pkg/gateway/models"
)

type resourceMessages struct {
	BindPage          string
	PageFailed        string
	PageSuccess       string
	BindCreate        string
	CreateFailed      string
	CreateSuccess     string
	BindUpdate        string
	UpdateFailed      string
	UpdateSuccess     string
	BindDelete        string
	DeleteFailed      string
	DeleteSuccess     string
	BindGet           string
	GetFailed         string
	GetSuccess        string
	GetAllFailed      string
	GetAllSuccess     string
	GetEnabledFailed  string
	GetEnabledSuccess string
}

type standardResourceHandler[TModel any, TQuery any, TPage any] struct {
	logger       *slog.Logger
	messages     resourceMessages
	page         func(TQuery) (TPage, error)
	create       func(TModel) error
	update       func(TModel) error
	delete       func(string) error
	get          func(string) (TModel, error)
	list         func() ([]TModel, error)
	listEnabled  func() ([]TModel, error)
	beforeCreate func(TModel) error
	beforeUpdate func(TModel) error
}

type crudResourceHandler[TModel any, TQuery any, TPage any, TGet any] struct {
	logger       *slog.Logger
	messages     resourceMessages
	page         func(TQuery) (TPage, error)
	create       func(TModel) error
	update       func(TModel) error
	delete       func(string) error
	get          func(string) (TGet, error)
	beforeCreate func(TModel) error
	beforeUpdate func(TModel) error
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TQuery](r)
	if err != nil {
		h.logger.Error(h.messages.BindPage, "error", err)
		http.Error(w, h.messages.BindPage, http.StatusBadRequest)
		return
	}

	result, err := h.page(req)
	if err != nil {
		h.logger.Error(h.messages.PageFailed, "error", err)
		http.Error(w, h.messages.PageFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TPage]{
		Code:    http.StatusOK,
		Message: h.messages.PageSuccess,
		Data:    result,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TModel](r)
	if err != nil {
		h.logger.Error(h.messages.BindCreate, "error", err)
		http.Error(w, h.messages.BindCreate, http.StatusBadRequest)
		return
	}

	if h.beforeCreate != nil {
		if err := h.beforeCreate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := h.create(req); err != nil {
		h.logger.Error(h.messages.CreateFailed, "error", err)
		http.Error(w, h.messages.CreateFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TModel]{
		Code:    http.StatusOK,
		Message: h.messages.CreateSuccess,
		Data:    req,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TModel](r)
	if err != nil {
		h.logger.Error(h.messages.BindUpdate, "error", err)
		http.Error(w, h.messages.BindUpdate, http.StatusBadRequest)
		return
	}

	if h.beforeUpdate != nil {
		if err := h.beforeUpdate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := h.update(req); err != nil {
		h.logger.Error(h.messages.UpdateFailed, "error", err)
		http.Error(w, h.messages.UpdateFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TModel]{
		Code:    http.StatusOK,
		Message: h.messages.UpdateSuccess,
		Data:    req,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error(h.messages.BindDelete, "error", err)
		http.Error(w, h.messages.BindDelete, http.StatusBadRequest)
		return
	}

	if err := h.delete(id); err != nil {
		h.logger.Error(h.messages.DeleteFailed, "error", err)
		http.Error(w, h.messages.DeleteFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: h.messages.DeleteSuccess,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error(h.messages.BindGet, "error", err)
		http.Error(w, h.messages.BindGet, http.StatusBadRequest)
		return
	}

	result, err := h.get(id)
	if err != nil {
		h.logger.Error(h.messages.GetFailed, "error", err)
		http.Error(w, h.messages.GetFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TModel]{
		Code:    http.StatusOK,
		Message: h.messages.GetSuccess,
		Data:    result,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) GetAll(w http.ResponseWriter, r *http.Request) {
	result, err := h.list()
	if err != nil {
		h.logger.Error(h.messages.GetAllFailed, "error", err)
		http.Error(w, h.messages.GetAllFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]TModel]{
		Code:    http.StatusOK,
		Message: h.messages.GetAllSuccess,
		Data:    result,
	})
}

func (h *standardResourceHandler[TModel, TQuery, TPage]) GetEnabled(w http.ResponseWriter, r *http.Request) {
	if h.listEnabled == nil {
		http.Error(w, "handler does not support enabled listing", http.StatusNotFound)
		return
	}

	result, err := h.listEnabled()
	if err != nil {
		h.logger.Error(h.messages.GetEnabledFailed, "error", err)
		http.Error(w, h.messages.GetEnabledFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[[]TModel]{
		Code:    http.StatusOK,
		Message: h.messages.GetEnabledSuccess,
		Data:    result,
	})
}

func (h *crudResourceHandler[TModel, TQuery, TPage, TGet]) Page(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TQuery](r)
	if err != nil {
		h.logger.Error(h.messages.BindPage, "error", err)
		http.Error(w, h.messages.BindPage, http.StatusBadRequest)
		return
	}

	result, err := h.page(req)
	if err != nil {
		h.logger.Error(h.messages.PageFailed, "error", err)
		http.Error(w, h.messages.PageFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TPage]{
		Code:    http.StatusOK,
		Message: h.messages.PageSuccess,
		Data:    result,
	})
}

func (h *crudResourceHandler[TModel, TQuery, TPage, TGet]) Create(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TModel](r)
	if err != nil {
		h.logger.Error(h.messages.BindCreate, "error", err)
		http.Error(w, h.messages.BindCreate, http.StatusBadRequest)
		return
	}

	if h.beforeCreate != nil {
		if err := h.beforeCreate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := h.create(req); err != nil {
		h.logger.Error(h.messages.CreateFailed, "error", err)
		http.Error(w, h.messages.CreateFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TModel]{
		Code:    http.StatusOK,
		Message: h.messages.CreateSuccess,
		Data:    req,
	})
}

func (h *crudResourceHandler[TModel, TQuery, TPage, TGet]) Update(w http.ResponseWriter, r *http.Request) {
	req, err := models.Bind[TModel](r)
	if err != nil {
		h.logger.Error(h.messages.BindUpdate, "error", err)
		http.Error(w, h.messages.BindUpdate, http.StatusBadRequest)
		return
	}

	if h.beforeUpdate != nil {
		if err := h.beforeUpdate(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := h.update(req); err != nil {
		h.logger.Error(h.messages.UpdateFailed, "error", err)
		http.Error(w, h.messages.UpdateFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TModel]{
		Code:    http.StatusOK,
		Message: h.messages.UpdateSuccess,
		Data:    req,
	})
}

func (h *crudResourceHandler[TModel, TQuery, TPage, TGet]) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error(h.messages.BindDelete, "error", err)
		http.Error(w, h.messages.BindDelete, http.StatusBadRequest)
		return
	}

	if err := h.delete(id); err != nil {
		h.logger.Error(h.messages.DeleteFailed, "error", err)
		http.Error(w, h.messages.DeleteFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[any]{
		Code:    http.StatusOK,
		Message: h.messages.DeleteSuccess,
	})
}

func (h *crudResourceHandler[TModel, TQuery, TPage, TGet]) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := models.BindID(r)
	if err != nil {
		h.logger.Error(h.messages.BindGet, "error", err)
		http.Error(w, h.messages.BindGet, http.StatusBadRequest)
		return
	}

	result, err := h.get(id)
	if err != nil {
		h.logger.Error(h.messages.GetFailed, "error", err)
		http.Error(w, h.messages.GetFailed, http.StatusInternalServerError)
		return
	}

	models.WriteData(w, models.BaseResponse[TGet]{
		Code:    http.StatusOK,
		Message: h.messages.GetSuccess,
		Data:    result,
	})
}
