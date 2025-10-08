package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bwburch/inflight-ui-service/internal/storage/simulations"
	"github.com/bwburch/inflight-ui-service/internal/storage/users"
	"github.com/labstack/echo/v4"
)

const (
	MaxFileSize      = 10 * 1024 * 1024 // 10 MB per file
	MaxTotalSize     = 50 * 1024 * 1024 // 50 MB total per job
	MaxFormMemory    = 32 * 1024 * 1024 // 32 MB in-memory buffer
)

type AttachmentsHandler struct {
	attachmentStore *simulations.S3AttachmentStore
	jobQueueStore   *simulations.JobQueueStore
}

func NewAttachmentsHandler(attachmentStore *simulations.S3AttachmentStore, jobQueueStore *simulations.JobQueueStore) *AttachmentsHandler {
	return &AttachmentsHandler{
		attachmentStore: attachmentStore,
		jobQueueStore:   jobQueueStore,
	}
}

// UploadAttachment handles file upload for a simulation job
// POST /api/v1/simulations/queue/:id/attachments
func (h *AttachmentsHandler) UploadAttachment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get user from context
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	// Parse job ID
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job ID")
	}

	// Verify job exists and belongs to user
	job, err := h.jobQueueStore.GetJob(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get job")
	}
	if job == nil {
		return echo.NewHTTPError(http.StatusNotFound, "job not found")
	}
	if job.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "job belongs to another user")
	}

	// Check total size limit
	currentSize, err := h.attachmentStore.GetTotalSizeForJob(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check total size")
	}
	if currentSize >= MaxTotalSize {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("total attachment size limit exceeded (%d MB)", MaxTotalSize/(1024*1024)))
	}

	// Parse multipart form
	if err := c.Request().ParseMultipartForm(MaxFormMemory); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to parse form data")
	}

	// Get file from form
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	defer file.Close()

	// Validate file size
	if header.Size > MaxFileSize {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, fmt.Sprintf("file too large (max %d MB)", MaxFileSize/(1024*1024)))
	}

	// Check if adding this file would exceed total limit
	if currentSize+header.Size > MaxTotalSize {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "adding this file would exceed total size limit")
	}

	// Get attachment type from form (optional)
	attachmentTypeStr := c.FormValue("attachment_type")
	if attachmentTypeStr == "" {
		attachmentTypeStr = "other"
	}
	attachmentType := simulations.AttachmentType(attachmentTypeStr)

	// Validate attachment type
	validTypes := []simulations.AttachmentType{
		simulations.AttachmentTypeScreenshot,
		simulations.AttachmentTypeConfig,
		simulations.AttachmentTypeLog,
		simulations.AttachmentTypeDocumentation,
		simulations.AttachmentTypeOther,
	}
	isValid := false
	for _, t := range validTypes {
		if attachmentType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid attachment type")
	}

	// Get description (optional)
	description := c.FormValue("description")
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	// Sanitize filename
	fileName := sanitizeFileName(header.Filename)

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(fileName)
	}

	// Save file to MinIO S3
	storagePath, bytesWritten, err := h.attachmentStore.SaveFile(ctx, jobID, user.ID, fileName, file, contentType, header.Size)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save file")
	}

	// Create attachment record
	attachment, err := h.attachmentStore.CreateAttachment(ctx, simulations.CreateAttachmentInput{
		SimulationJobID: jobID,
		UserID:          user.ID,
		FileName:        fileName,
		FileType:        contentType,
		FileSize:        int(bytesWritten),
		StoragePath:     storagePath,
		AttachmentType:  attachmentType,
		Description:     descPtr,
	})

	if err != nil {
		// TODO: Clean up S3 object if database insert fails
		// For now, orphaned S3 objects can be cleaned up by background job
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create attachment record")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"attachment": attachment,
	})
}

// ListAttachments lists all attachments for a simulation job
// GET /api/v1/simulations/queue/:id/attachments
func (h *AttachmentsHandler) ListAttachments(c echo.Context) error {
	ctx := c.Request().Context()

	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job ID")
	}

	attachments, err := h.attachmentStore.ListAttachments(ctx, jobID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list attachments")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"attachments": attachments,
		"total":       len(attachments),
	})
}

// DownloadAttachment serves an attachment file from S3
// GET /api/v1/simulations/queue/:id/attachments/:attachmentId
func (h *AttachmentsHandler) DownloadAttachment(c echo.Context) error {
	ctx := c.Request().Context()

	attachmentID, err := strconv.Atoi(c.Param("attachmentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid attachment ID")
	}

	// Get attachment
	attachment, err := h.attachmentStore.GetAttachment(ctx, attachmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get attachment")
	}
	if attachment == nil {
		return echo.NewHTTPError(http.StatusNotFound, "attachment not found")
	}

	// Get file reader from S3
	reader, err := h.attachmentStore.GetFileReader(ctx, attachment)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve file from storage")
	}
	defer reader.Close()

	// Set headers
	c.Response().Header().Set("Content-Type", attachment.FileType)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.FileName))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", attachment.FileSize))

	// Stream file from S3 to response
	c.Response().WriteHeader(http.StatusOK)
	if _, err := io.Copy(c.Response().Writer, reader); err != nil {
		return fmt.Errorf("stream file: %w", err)
	}

	return nil
}

// DeleteAttachment deletes an attachment
// DELETE /api/v1/simulations/queue/:id/attachments/:attachmentId
func (h *AttachmentsHandler) DeleteAttachment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get user from context
	user, ok := c.Get("user").(*users.User)
	if !ok || user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
	}

	attachmentID, err := strconv.Atoi(c.Param("attachmentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid attachment ID")
	}

	// Get attachment to verify ownership
	attachment, err := h.attachmentStore.GetAttachment(ctx, attachmentID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get attachment")
	}
	if attachment == nil {
		return echo.NewHTTPError(http.StatusNotFound, "attachment not found")
	}
	if attachment.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "attachment belongs to another user")
	}

	// Delete attachment
	if err := h.attachmentStore.DeleteAttachment(ctx, attachmentID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete attachment")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "attachment deleted",
	})
}

// RegisterRoutes registers all attachment routes
func (h *AttachmentsHandler) RegisterRoutes(e *echo.Group, authMiddleware echo.MiddlewareFunc) {
	e.POST("/queue/:id/attachments", h.UploadAttachment, authMiddleware)
	e.GET("/queue/:id/attachments", h.ListAttachments, authMiddleware)
	e.GET("/queue/:id/attachments/:attachmentId", h.DownloadAttachment, authMiddleware)
	e.DELETE("/queue/:id/attachments/:attachmentId", h.DeleteAttachment, authMiddleware)
}

// Helper functions

// sanitizeFileName removes dangerous characters from filenames
func sanitizeFileName(fileName string) string {
	// Remove path separators
	fileName = filepath.Base(fileName)

	// Replace dangerous characters
	fileName = strings.ReplaceAll(fileName, "..", "")
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")

	// Limit length
	if len(fileName) > 255 {
		ext := filepath.Ext(fileName)
		baseName := fileName[:255-len(ext)]
		fileName = baseName + ext
	}

	return fileName
}

// detectContentType guesses content type from file extension
func detectContentType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))

	contentTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".json": "application/json",
		".txt":  "text/plain",
		".log":  "text/plain",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".md":   "text/markdown",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}
