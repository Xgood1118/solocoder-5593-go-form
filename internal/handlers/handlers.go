package handlers

import (
	"fmt"
	"net/http"
	"time"

	"dynamic-form-engine/internal/models"
	"dynamic-form-engine/internal/schema"
	"dynamic-form-engine/internal/storage"
	"dynamic-form-engine/internal/upload"
	"dynamic-form-engine/internal/validation"
	"dynamic-form-engine/internal/workflow"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store      *storage.Store
	workflow   *workflow.Service
	upload     *upload.Service
}

func NewHandler(store *storage.Store, wf *workflow.Service, up *upload.Service) *Handler {
	return &Handler{
		store:    store,
		workflow: wf,
		upload:   up,
	}
}

type CreateSchemaRequest struct {
	Name        string              `json:"name" binding:"required"`
	Description string              `json:"description"`
	Fields      []models.FieldDef   `json:"fields" binding:"required"`
	Workflow    *models.WorkflowConfig `json:"workflow"`
	SubmissionRateLimit *models.RateLimitConfig `json:"submission_rate_limit"`
}

func (h *Handler) CreateSchema(c *gin.Context) {
	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := &models.FormSchema{
		Name:                req.Name,
		Description:         req.Description,
		Fields:              req.Fields,
		Workflow:            req.Workflow,
		SubmissionRateLimit: req.SubmissionRateLimit,
	}

	if err := schema.ValidateSchema(s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schema 不合法: " + err.Error()})
		return
	}

	if err := h.store.CreateSchema(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, s)
}

func (h *Handler) UpdateSchema(c *gin.Context) {
	id := c.Param("id")

	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.store.GetSchema(id, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema 不存在"})
		return
	}

	s := &models.FormSchema{
		ID:                  id,
		Name:                req.Name,
		Description:         req.Description,
		Fields:              req.Fields,
		Workflow:            req.Workflow,
		SubmissionRateLimit: req.SubmissionRateLimit,
	}

	if err := schema.ValidateSchema(s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schema 不合法: " + err.Error()})
		return
	}

	if err := h.store.UpdateSchema(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updated, _ := h.store.GetSchema(id, 0)
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) GetSchema(c *gin.Context) {
	id := c.Param("id")
	versionStr := c.Query("version")

	version := 0
	if versionStr != "" {
		var err error
		_, err = fmt.Sscanf(versionStr, "%d", &version)
		if err != nil {
			version = 0
		}
	}

	s, err := h.store.GetSchema(id, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema 不存在"})
		return
	}

	c.JSON(http.StatusOK, s)
}

func (h *Handler) ListSchemas(c *gin.Context) {
	schemas, err := h.store.ListSchemas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if schemas == nil {
		schemas = []models.FormSchema{}
	}

	c.JSON(http.StatusOK, schemas)
}

func (h *Handler) DeleteSchema(c *gin.Context) {
	id := c.Param("id")

	deleted, err := h.store.DeleteSchema(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema 不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *Handler) ValidateSchema(c *gin.Context) {
	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := &models.FormSchema{
		Name:                req.Name,
		Description:         req.Description,
		Fields:              req.Fields,
		Workflow:            req.Workflow,
		SubmissionRateLimit: req.SubmissionRateLimit,
	}

	if err := schema.ValidateSchema(s); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

type SubmitRequest struct {
	SubmitterID   string                 `json:"submitter_id" binding:"required"`
	SubmitterName string                 `json:"submitter_name"`
	Data          map[string]interface{} `json:"data" binding:"required"`
}

func (h *Handler) SubmitForm(c *gin.Context) {
	formID := c.Param("id")

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s, err := h.store.GetSchema(formID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "表单不存在"})
		return
	}

	if s.SubmissionRateLimit != nil && s.SubmissionRateLimit.MaxPerMinute > 0 {
		allowed, err := h.store.RecordSubmissionAttempt(formID, req.SubmitterID, time.Minute, s.SubmissionRateLimit.MaxPerMinute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "提交过于频繁，请稍后再试",
				"code":  "rate_limited",
			})
			return
		}
	}

	result := validation.ValidateSubmission(s, req.Data)
	if !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":  false,
			"errors": result.Errors,
		})
		return
	}

	sub := &models.Submission{
		FormID:        formID,
		SchemaVersion: s.Version,
		SubmitterID:   req.SubmitterID,
		SubmitterName: req.SubmitterName,
		Data:          req.Data,
		Status:        models.SubmissionStatusDraft,
	}

	if s.Workflow != nil && s.Workflow.Enabled {
		if err := h.workflow.StartWorkflow(sub, s); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		sub.Status = models.SubmissionStatusApproved
	}

	if err := h.store.CreateSubmission(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) ValidateSubmission(c *gin.Context) {
	formID := c.Param("id")

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s, err := h.store.GetSchema(formID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "表单不存在"})
		return
	}

	result := validation.ValidateSubmission(s, req.Data)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetSubmission(c *gin.Context) {
	id := c.Param("id")

	sub, err := h.store.GetSubmission(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "提交记录不存在"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

func (h *Handler) ListSubmissions(c *gin.Context) {
	formID := c.Query("form_id")

	subs, err := h.store.ListSubmissions(formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if subs == nil {
		subs = []models.Submission{}
	}

	c.JSON(http.StatusOK, subs)
}

type ApprovalRequest struct {
	Approver string `json:"approver" binding:"required"`
	Comment  string `json:"comment"`
}

func (h *Handler) ApproveSubmission(c *gin.Context) {
	id := c.Param("id")

	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		sub, err := h.store.GetSubmission(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if sub == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "提交记录不存在"})
			return
		}

		if err := h.workflow.Approve(sub, req.Approver, req.Comment); err != nil {
			if err == models.ErrConflict {
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, sub)
		return
	}

	c.JSON(http.StatusConflict, gin.H{"error": "操作冲突，请重试", "code": "conflict"})
}

func (h *Handler) RejectSubmission(c *gin.Context) {
	id := c.Param("id")

	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		sub, err := h.store.GetSubmission(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if sub == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "提交记录不存在"})
			return
		}

		if err := h.workflow.Reject(sub, req.Approver, req.Comment); err != nil {
			if err == models.ErrConflict {
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, sub)
		return
	}

	c.JSON(http.StatusConflict, gin.H{"error": "操作冲突，请重试", "code": "conflict"})
}

func (h *Handler) UploadFile(c *gin.Context) {
	formID := c.Param("id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	relPath, size, err := h.upload.UploadFile(formID, file.Filename, src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"path": relPath,
		"url":  h.upload.GetFileURL(relPath),
		"size": size,
		"name": file.Filename,
	})
}

func (h *Handler) DeleteSubmission(c *gin.Context) {
	id := c.Param("id")

	deleted, err := h.store.DeleteSubmission(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "提交记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
