package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
	"matcha/api/internal/validation"
)

type ProfileHandler struct {
	profileRepo *repository.ProfileRepository
	photoRepo   *repository.PhotoRepository
	syncSvc     *services.SyncService
}

func NewProfileHandler(profileRepo *repository.ProfileRepository, photoRepo *repository.PhotoRepository, syncSvc *services.SyncService) *ProfileHandler {
	return &ProfileHandler{profileRepo: profileRepo, photoRepo: photoRepo, syncSvc: syncSvc}
}

type UpdateProfileReq struct {
	Bio              *string  `json:"bio"`               // max 500 chars
	Gender           *string  `json:"gender"`            // male, female, non-binary, other
	SexualPreference *string  `json:"sexual_preference"` // male, female, both, other
	BirthDate        *string  `json:"birth_date"`        // YYYY-MM-DD, past, 18+
	Latitude         *float64 `json:"latitude"`          // -90 to 90
	Longitude        *float64 `json:"longitude"`         // -180 to 180
}

// GetMe godoc
// @Summary	Get own profile
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Success	200	{object}	map[string]interface{}
// @Failure	401	{object}	map[string]string
// @Router		/api/v1/profile/me [get]
func (h *ProfileHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	photos, _ := h.photoRepo.ListByUser(c.Request.Context(), id)

	p, err := h.profileRepo.GetByUserID(c.Request.Context(), id)
	if err != nil {
		resp := gin.H{
			"user_id":     id,
			"fame_rating": 0,
		}
		attachPhotos(resp, photos)
		c.JSON(http.StatusOK, resp)
		return
	}
	resp := toProfileResp(p)
	attachPhotos(resp, photos)
	c.JSON(http.StatusOK, resp)
}

// UpdateMe godoc
// @Summary	Update own profile
// @Tags		profile
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		body	body		UpdateProfileReq	true	"Profile fields"
// @Success	200	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Failure	401	{object}	map[string]string
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/profile/me [put]
func (h *ProfileHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	var req UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validation
	if req.BirthDate != nil && *req.BirthDate != "" {
		if err := validation.ValidateBirthDate(*req.BirthDate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Gender != nil && *req.Gender != "" {
		if err := validation.ValidateGender(*req.Gender); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.SexualPreference != nil && *req.SexualPreference != "" {
		if err := validation.ValidateSexualPreference(*req.SexualPreference); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Bio != nil {
		if err := validation.ValidateBio(*req.Bio); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Latitude != nil {
		if err := validation.ValidateLatitude(*req.Latitude); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Longitude != nil {
		if err := validation.ValidateLongitude(*req.Longitude); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
	if err := h.syncSvc.SyncUser(c.Request.Context(), id); err != nil {
		log.Printf("[profile] sync to ES failed for user=%s: %v", id, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func toProfileResp(p *repository.Profile) gin.H {
	resp := gin.H{
		"user_id":     p.UserID,
		"fame_rating": p.FameRating,
		"created_at":  p.CreatedAt,
		"updated_at":  p.UpdatedAt,
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

func attachPhotos(resp gin.H, photos []repository.Photo) {
	photoResp := make([]gin.H, len(photos))
	for i := range photos {
		photoResp[i] = gin.H{
			"id":         photos[i].ID,
			"url":        photos[i].URL,
			"is_primary": photos[i].IsPrimary,
			"position":   photos[i].Position,
		}
		if photos[i].IsPrimary {
			resp["primary_photo_url"] = photos[i].URL
		}
	}
	resp["photos"] = photoResp
}
