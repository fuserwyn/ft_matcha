package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
	"matcha/api/internal/storage"
	"matcha/api/internal/validation"
)

type ProfileHandler struct {
	profileRepo   *repository.ProfileRepository
	photoRepo     *repository.PhotoRepository
	discoveryRepo *repository.DiscoveryRepository
	syncSvc       *services.SyncService
	photoStore    *storage.MinIO
	apiBaseURL    string
}

func NewProfileHandler(profileRepo *repository.ProfileRepository, photoRepo *repository.PhotoRepository, discoveryRepo *repository.DiscoveryRepository, syncSvc *services.SyncService, photoStore *storage.MinIO, apiBaseURL string) *ProfileHandler {
	return &ProfileHandler{profileRepo: profileRepo, photoRepo: photoRepo, discoveryRepo: discoveryRepo, syncSvc: syncSvc, photoStore: photoStore, apiBaseURL: strings.TrimRight(apiBaseURL, "/")}
}

type UpdateProfileReq struct {
	Bio              *string  `json:"bio"`               // max 500 chars
	Gender           *string  `json:"gender"`            // male, female, non-binary, other
	SexualPreference *[]string `json:"sexual_preference"` // array: male, female, non-binary, other
	RelationshipGoal *string  `json:"relationship_goal"` // long-term, long-term-open, short-term-open, short-term, friends, not-sure
	BirthDate        *string  `json:"birth_date"`        // YYYY-MM-DD, past, 18+
	City             *string  `json:"city"`              // manually entered city
	Latitude         *float64 `json:"latitude"`          // -90 to 90
	Longitude        *float64 `json:"longitude"`         // -180 to 180
}

type UpdateTagsReq struct {
	Tags []string `json:"tags" binding:"required"`
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
		if tags, tagsErr := h.profileRepo.GetTags(c.Request.Context(), id); tagsErr == nil {
			resp["tags"] = tags
		}
		h.attachPhotos(resp, photos)
		c.JSON(http.StatusOK, resp)
		return
	}
	resp := toProfileResp(p)
	if tags, tagsErr := h.profileRepo.GetTags(c.Request.Context(), id); tagsErr == nil {
		resp["tags"] = tags
	}
	h.attachPhotos(resp, photos)
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
	sexualPreference := normalizeSexualPreference(req.SexualPreference)
	if len(sexualPreference) > 0 {
		if err := validation.ValidateSexualPreference(sexualPreference); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.RelationshipGoal != nil && *req.RelationshipGoal != "" {
		if err := validation.ValidateRelationshipGoal(*req.RelationshipGoal); err != nil {
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
	if req.City != nil {
		if err := validation.ValidateCity(*req.City); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	p := &repository.Profile{
		UserID:           id,
		Bio:              req.Bio,
		Gender:           req.Gender,
		SexualPreference: sexualPreference,
		RelationshipGoal: req.RelationshipGoal,
		City:             req.City,
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

// GetMyTags godoc
// @Summary	Get own tags
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Success	200	{object}	map[string]interface{}
// @Router		/api/v1/profile/me/tags [get]
func (h *ProfileHandler) GetMyTags(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	tags, err := h.profileRepo.GetTags(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// UpdateMyTags godoc
// @Summary	Update own tags
// @Tags		profile
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		body	body		UpdateTagsReq	true	"Tags payload"
// @Success	200	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Router		/api/v1/profile/me/tags [put]
func (h *ProfileHandler) UpdateMyTags(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	var req UpdateTagsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	normalized := make([]string, 0, len(req.Tags))
	seen := make(map[string]struct{}, len(req.Tags))
	for _, tag := range req.Tags {
		t := strings.ToLower(strings.TrimSpace(tag))
		t = strings.TrimPrefix(t, "#")
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		normalized = append(normalized, t)
	}
	if err := validation.ValidateTags(normalized); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.profileRepo.SetTags(c.Request.Context(), id, normalized); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.syncSvc.SyncUser(c.Request.Context(), id); err != nil {
		log.Printf("[profile] sync to ES failed for user=%s: %v", id, err)
	}
	c.JSON(http.StatusOK, gin.H{"tags": normalized})
}

// GetViewedHistory godoc
// @Summary	Get profiles I viewed
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200	{array}		object
// @Router		/api/v1/profile/me/views [get]
func (h *ProfileHandler) GetViewedHistory(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	history, err := h.profileRepo.GetViewedProfiles(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]gin.H, len(history))
	for i := range history {
		resp[i] = gin.H{
			"id":             history[i].UserID,
			"username":       history[i].Username,
			"first_name":     history[i].FirstName,
			"last_name":      history[i].LastName,
			"fame_rating":    history[i].FameRating,
			"last_viewed_at": history[i].LastViewedAt,
		}
		if history[i].City != nil {
			resp[i]["city"] = *history[i].City
		}
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), history[i].UserID); err == nil && p != nil {
			resp[i]["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
	}
	c.JSON(http.StatusOK, resp)
}

// GetViewedMe godoc
// @Summary	Get profiles who viewed me
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200	{array}		object
// @Router		/api/v1/profile/me/viewed-by [get]
func (h *ProfileHandler) GetViewedMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	history, err := h.profileRepo.GetProfilesWhoViewedMe(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]gin.H, len(history))
	for i := range history {
		resp[i] = gin.H{
			"id":             history[i].UserID,
			"username":       history[i].Username,
			"first_name":     history[i].FirstName,
			"last_name":      history[i].LastName,
			"fame_rating":    history[i].FameRating,
			"last_viewed_at": history[i].LastViewedAt,
		}
		if history[i].City != nil {
			resp[i]["city"] = *history[i].City
		}
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), history[i].UserID); err == nil && p != nil {
			resp[i]["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
	}
	c.JSON(http.StatusOK, resp)
}

// TagSuggestions godoc
// @Summary	Get tag suggestions (top tags or prefix match, e.g. ?q=mus -> music)
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Param		q	query		string	false	"Tag prefix for autocomplete"
// @Success	200	{object}	map[string]interface{}
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/profile/tags/suggestions [get]
func (h *ProfileHandler) TagSuggestions(c *gin.Context) {
	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	if q != "" {
		tags, err := h.discoveryRepo.SearchTags(c.Request.Context(), q, 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tags": tags})
		return
	}
	tags, err := h.profileRepo.ListTopTags(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// CitySuggestions godoc
// @Summary	Get city suggestions (partial match, e.g. "Par" -> Paris)
// @Tags		profile
// @Security	BearerAuth
// @Produce	json
// @Param		q	query		string	false	"City prefix"
// @Success	200	{object}	map[string]interface{}
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/profile/cities/suggestions [get]
func (h *ProfileHandler) CitySuggestions(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	cities, err := h.discoveryRepo.SearchCities(c.Request.Context(), q, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cities": cities})
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
	if len(p.SexualPreference) > 0 {
		resp["sexual_preference"] = p.SexualPreference
	}
	if p.RelationshipGoal != nil {
		resp["relationship_goal"] = *p.RelationshipGoal
	}
	if p.BirthDate != nil {
		resp["birth_date"] = p.BirthDate.Format("2006-01-02")
	}
	if p.City != nil {
		resp["city"] = *p.City
	}
	if p.Latitude != nil {
		resp["latitude"] = *p.Latitude
	}
	if p.Longitude != nil {
		resp["longitude"] = *p.Longitude
	}
	return resp
}

func (h *ProfileHandler) attachPhotos(resp gin.H, photos []repository.Photo) {
	photoResp := make([]gin.H, len(photos))
	for i := range photos {
		url := photoURL(&photos[i], h.apiBaseURL)
		photoResp[i] = gin.H{
			"id":         photos[i].ID,
			"url":        url,
			"is_primary": photos[i].IsPrimary,
			"position":   photos[i].Position,
		}
		if photos[i].IsPrimary {
			resp["primary_photo_url"] = url
		}
	}
	resp["photos"] = photoResp
}

func normalizeSexualPreference(values *[]string) []string {
	if values == nil {
		return nil
	}
	if len(*values) == 0 {
		return []string{"male", "female"}
	}
	out := make([]string, len(*values))
	copy(out, *values)
	return out
}
