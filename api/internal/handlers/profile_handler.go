package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type ProfileHandler struct {
	profileRepo *repository.ProfileRepository
}

func NewProfileHandler(profileRepo *repository.ProfileRepository) *ProfileHandler {
	return &ProfileHandler{profileRepo: profileRepo}
}

type UpdateProfileReq struct {
	Bio              *string  `json:"bio"`
	Gender           *string  `json:"gender"`
	SexualPreference *string  `json:"sexual_preference"`
	BirthDate        *string  `json:"birth_date"` // YYYY-MM-DD
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
}

func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	p, err := h.profileRepo.GetByUserID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"user_id":      id,
			"fame_rating":  0,
		})
		return
	}
	c.JSON(http.StatusOK, toProfileResp(p))
}

func (h *ProfileHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	var req UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := &repository.Profile{
		UserID:           id,
		Bio:              req.Bio,
		Gender:           req.Gender,
		SexualPreference: req.SexualPreference,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		FameRating:       0,
	}
	if req.BirthDate != nil && *req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", *req.BirthDate)
		if err == nil {
			p.BirthDate = &t
		}
	}
	if err := h.profileRepo.Upsert(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func toProfileResp(p *repository.Profile) gin.H {
	resp := gin.H{
		"user_id":      p.UserID,
		"fame_rating":  p.FameRating,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
	if p.Bio != nil {
		resp["bio"] = *p.Bio
	}
	if p.Gender != nil {
		resp["gender"] = *p.Gender
	}
	if p.SexualPreference != nil {
		resp["sexual_preference"] = *p.SexualPreference
	}
	if p.BirthDate != nil {
		resp["birth_date"] = p.BirthDate.Format("2006-01-02")
	}
	if p.Latitude != nil {
		resp["latitude"] = *p.Latitude
	}
	if p.Longitude != nil {
		resp["longitude"] = *p.Longitude
	}
	return resp
}
