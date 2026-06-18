package main

import (
	"log"
	"time"

	"dynamic-form-engine/internal/handlers"
	"dynamic-form-engine/internal/storage"
	"dynamic-form-engine/internal/upload"
	"dynamic-form-engine/internal/workflow"

	"github.com/gin-gonic/gin"
)

func main() {
	store, err := storage.NewStore("form.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	wf := workflow.NewService(store)

	uploadSvc, err := upload.NewService("uploads")
	if err != nil {
		log.Fatalf("Failed to init upload service: %v", err)
	}

	h := handlers.NewHandler(store, wf, uploadSvc)
	admin := handlers.NewAdminHandler(h, "templates")

	r := gin.Default()

	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	{
		schemas := api.Group("/schemas")
		{
			schemas.POST("", h.CreateSchema)
			schemas.GET("", h.ListSchemas)
			schemas.GET("/:id", h.GetSchema)
			schemas.PUT("/:id", h.UpdateSchema)
			schemas.DELETE("/:id", h.DeleteSchema)
			schemas.POST("/validate", h.ValidateSchema)
		}

		forms := api.Group("/forms")
		{
			forms.POST("/:id/submit", h.SubmitForm)
			forms.POST("/:id/validate", h.ValidateSubmission)
			forms.POST("/:id/upload", h.UploadFile)
		}

		submissions := api.Group("/submissions")
		{
			submissions.GET("", h.ListSubmissions)
			submissions.GET("/:id", h.GetSubmission)
			submissions.DELETE("/:id", h.DeleteSubmission)
			submissions.POST("/:id/approve", h.ApproveSubmission)
			submissions.POST("/:id/reject", h.RejectSubmission)
		}
	}

	adminGroup := r.Group("/admin")
	{
		adminGroup.GET("", admin.Index)
		adminGroup.GET("/schemas", admin.ListSchemas)
		adminGroup.GET("/schemas/:id", admin.GetSchema)
		adminGroup.GET("/submissions", admin.ListSubmissions)
		adminGroup.GET("/submissions/:id", admin.GetSubmission)
	}

	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/admin")
	})

	go cleanupRateLimit(store)

	log.Println("Dynamic Form Engine starting on :8143")
	log.Println("Admin panel: http://localhost:8143/admin")
	log.Println("API docs:")
	log.Println("  GET  /api/schemas          - List all schemas")
	log.Println("  POST /api/schemas          - Create schema")
	log.Println("  GET  /api/schemas/:id      - Get schema")
	log.Println("  PUT  /api/schemas/:id      - Update schema (new version)")
	log.Println("  POST /api/schemas/validate - Validate schema")
	log.Println("  POST /api/forms/:id/submit - Submit form data")
	log.Println("  POST /api/forms/:id/validate - Validate form data")
	log.Println("  POST /api/forms/:id/upload - Upload file")
	log.Println("  GET  /api/submissions      - List submissions")
	log.Println("  GET  /api/submissions/:id  - Get submission")
	log.Println("  POST /api/submissions/:id/approve - Approve submission")
	log.Println("  POST /api/submissions/:id/reject  - Reject submission")

	if err := r.Run(":8143"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func cleanupRateLimit(store *storage.Store) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-24 * time.Hour)
		store.CleanupOldRateLimit(cutoff)
	}
}
