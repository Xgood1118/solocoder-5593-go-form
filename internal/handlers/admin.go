package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"dynamic-form-engine/internal/models"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	handler *Handler
	tplDir  string
}

func NewAdminHandler(h *Handler, tplDir string) *AdminHandler {
	return &AdminHandler{
		handler: h,
		tplDir:  tplDir,
	}
}

func (ah *AdminHandler) loadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"jsonMarshal": func(v interface{}) string {
			data, _ := json.MarshalIndent(v, "", "  ")
			return string(data)
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tpl := template.New("").Funcs(funcMap)

	files, err := filepath.Glob(filepath.Join(ah.tplDir, "*.html"))
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		_, err := tpl.ParseFiles(f)
		if err != nil {
			return nil, err
		}
	}

	return tpl, nil
}

func (ah *AdminHandler) Index(c *gin.Context) {
	tpl, err := ah.loadTemplates()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	schemas, _ := ah.handler.store.ListSchemas()
	submissions, _ := ah.handler.store.ListSubmissions("")

	pendingCount := 0
	approvedCount := 0
	rejectedCount := 0
	for _, s := range submissions {
		switch s.Status {
		case models.SubmissionStatusPending:
			pendingCount++
		case models.SubmissionStatusApproved:
			approvedCount++
		case models.SubmissionStatusRejected:
			rejectedCount++
		}
	}

	data := gin.H{
		"Title":         "首页",
		"SchemaCount":   len(schemas),
		"PendingCount":  pendingCount,
		"ApprovedCount": approvedCount,
		"RejectedCount": rejectedCount,
	}

	tpl.ExecuteTemplate(c.Writer, "index.html", data)
}

func (ah *AdminHandler) ListSchemas(c *gin.Context) {
	tpl, err := ah.loadTemplates()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	schemas, _ := ah.handler.store.ListSchemas()

	data := gin.H{
		"Title":   "表单管理",
		"Schemas": schemas,
	}

	tpl.ExecuteTemplate(c.Writer, "schemas.html", data)
}

func (ah *AdminHandler) GetSchema(c *gin.Context) {
	tpl, err := ah.loadTemplates()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	id := c.Param("id")
	schema, _ := ah.handler.store.GetSchema(id, 0)

	if schema == nil {
		c.String(http.StatusNotFound, "表单不存在")
		return
	}

	data := gin.H{
		"Title":  schema.Name,
		"Schema": schema,
	}

	tpl.ExecuteTemplate(c.Writer, "schema_detail.html", data)
}

func (ah *AdminHandler) ListSubmissions(c *gin.Context) {
	tpl, err := ah.loadTemplates()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	formID := c.Query("form_id")
	submissions, _ := ah.handler.store.ListSubmissions(formID)

	data := gin.H{
		"Title":       "提交记录",
		"Submissions": submissions,
		"FormID":      formID,
	}

	tpl.ExecuteTemplate(c.Writer, "submissions.html", data)
}

func (ah *AdminHandler) GetSubmission(c *gin.Context) {
	tpl, err := ah.loadTemplates()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	id := c.Param("id")
	sub, _ := ah.handler.store.GetSubmission(id)

	if sub == nil {
		c.String(http.StatusNotFound, "提交记录不存在")
		return
	}

	schema, _ := ah.handler.store.GetSchema(sub.FormID, sub.SchemaVersion)

	data := gin.H{
		"Title":      "提交详情",
		"Submission": sub,
		"Schema":     schema,
	}

	tpl.ExecuteTemplate(c.Writer, "submission_detail.html", data)
}
