package handlers

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/storage"
)

const maxPhotosPerUser = 5

var allowedPhotoContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var allowedPhotoExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

type PhotoHandler struct {
	photos *repository.PhotoRepository
	store  *storage.MinIO
}

func NewPhotoHandler(photos *repository.PhotoRepository, store *storage.MinIO) *PhotoHandler {
	return &PhotoHandler{photos: photos, store: store}
}

// UploadMe godoc
// @Summary	Upload own photo
// @Tags		photos
// @Security	BearerAuth
// @Accept		multipart/form-data
// @Produce	json
// @Param		file	formData	file	true	"Photo file"
// @Success	201	{object}	object
// @Router		/api/v1/photos [post]
func (h *PhotoHandler) UploadMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	existing, err := h.photos.ListByUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(existing) >= maxPhotosPerUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max 5 photos allowed"})
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	if fh.Size <= 0 || fh.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be 1B..10MB"})
		return
	}
	contentType := strings.TrimSpace(fh.Header.Get("Content-Type"))
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fh.Filename)))
	if !allowedPhotoExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed file extensions: .jpg, .jpeg, .png, .webp"})
		return
	}

	file, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
		return
	}
	defer file.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	detectedType := http.DetectContentType(head[:n])
	if !allowedPhotoContentTypes[detectedType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed file types: image/jpeg, image/png, image/webp"})
		return
	}
	if contentType != "" && !allowedPhotoContentTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content-type header"})
		return
	}

	reader := io.MultiReader(bytes.NewReader(head[:n]), file)
	objectKey := storage.BuildPhotoObjectKey(id.String(), uuid.NewString(), fh.Filename)
	url, err := h.store.PutObject(c.Request.Context(), objectKey, reader, fh.Size, detectedType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	makePrimary := len(existing) == 0
	p, err := h.photos.Create(c.Request.Context(), id, objectKey, url, makePrimary)
	if err != nil {
		_ = h.store.RemoveObject(c.Request.Context(), objectKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, photoResp(p))
}

// ListMe godoc
// @Summary	List own photos
// @Tags		photos
// @Security	BearerAuth
// @Produce	json
// @Success	200	{array}	object
// @Router		/api/v1/photos/me [get]
func (h *PhotoHandler) ListMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	items, err := h.photos.ListByUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]gin.H, len(items))
	for i := range items {
		resp[i] = photoResp(&items[i])
	}
	c.JSON(http.StatusOK, resp)
}

// ListByUser godoc
// @Summary	List photos by user id
// @Tags		photos
// @Security	BearerAuth
// @Produce	json
// @Param		id	path		string	true	"User ID"
// @Success	200	{array}	object
// @Router		/api/v1/users/{id}/photos [get]
func (h *PhotoHandler) ListByUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	items, err := h.photos.ListByUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]gin.H, len(items))
	for i := range items {
		resp[i] = photoResp(&items[i])
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteMe godoc
// @Summary	Delete own photo
// @Tags		photos
// @Security	BearerAuth
// @Produce	json
// @Param		id	path		string	true	"Photo ID"
// @Success	204
// @Router		/api/v1/photos/{id} [delete]
func (h *PhotoHandler) DeleteMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	photoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	p, err := h.photos.GetByID(c.Request.Context(), photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil || p.UserID != id {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}

	if err := h.photos.DeleteByID(c.Request.Context(), id, photoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.RemoveObject(c.Request.Context(), p.ObjectKey)
	c.Status(http.StatusNoContent)
}

// SetPrimaryMe godoc
// @Summary	Set own primary photo
// @Tags		photos
// @Security	BearerAuth
// @Produce	json
// @Param		id	path		string	true	"Photo ID"
// @Success	200	{object}	object
// @Router		/api/v1/photos/{id}/primary [patch]
func (h *PhotoHandler) SetPrimaryMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	photoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.photos.GetByID(c.Request.Context(), photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil || p.UserID != id {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}

	if err := h.photos.SetPrimary(c.Request.Context(), id, photoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func photoResp(p *repository.Photo) gin.H {
	return gin.H{
		"id":         p.ID,
		"user_id":    p.UserID,
		"url":        p.URL,
		"is_primary": p.IsPrimary,
		"position":   p.Position,
		"created_at": p.CreatedAt,
	}
}
