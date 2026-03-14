package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
	"matcha/api/internal/storage"
	ws "matcha/api/internal/websocket"
)

type DiscoveryHandler struct {
	userRepo      *repository.UserRepository
	profileRepo   *repository.ProfileRepository
	photoRepo     *repository.PhotoRepository
	likeRepo      *repository.LikeRepository
	blockRepo     *repository.BlockRepository
	notifRepo     *repository.NotificationRepository
	discoveryRepo *repository.DiscoveryRepository
	syncSvc       *services.SyncService
	hub           *ws.Hub
	photoStore    *storage.MinIO
	apiBaseURL    string
}

func NewDiscoveryHandler(
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	photoRepo *repository.PhotoRepository,
	likeRepo *repository.LikeRepository,
	blockRepo *repository.BlockRepository,
	notifRepo *repository.NotificationRepository,
	discoveryRepo *repository.DiscoveryRepository,
	syncSvc *services.SyncService,
	hub *ws.Hub,
	photoStore *storage.MinIO,
	apiBaseURL string,
) *DiscoveryHandler {
	return &DiscoveryHandler{
		userRepo:      userRepo,
		profileRepo:   profileRepo,
		photoRepo:     photoRepo,
		likeRepo:      likeRepo,
		blockRepo:     blockRepo,
		notifRepo:     notifRepo,
		discoveryRepo: discoveryRepo,
		syncSvc:       syncSvc,
		hub:           hub,
		photoStore:    photoStore,
		apiBaseURL:    strings.TrimRight(apiBaseURL, "/"),
	}
}

// FilterAggregations godoc
// @Summary	Get filter counts (gender, interest) for discovery
// @Tags		discovery
// @Security	BearerAuth
// @Produce	json
// @Success	200	{object}	map[string]interface{}
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/users/filters/aggregations [get]
func (h *DiscoveryHandler) FilterAggregations(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	excludeIDs := []uuid.UUID{}
	if blocked, err := h.blockRepo.ListBlockedIDs(c.Request.Context(), id); err == nil {
		excludeIDs = blocked
	}
	gender, interest, relationshipGoal, err := h.discoveryRepo.FilterAggregations(c.Request.Context(), id, excludeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gender":            gender,
		"interest":          interest,
		"relationship_goal": relationshipGoal,
	})
}

// Search godoc
// @Summary	Search users
// @Tags		discovery
// @Security	BearerAuth
// @Produce	json
// @Param		gender		query		string	false	"Filter by gender"
// @Param		interest	query		string	false	"Filter by interest (sexual_preference)"
// @Param		min_age		query		int		false	"Min age"
// @Param		max_age		query		int		false	"Max age"
// @Param		limit		query		int		false	"Limit (default 20)"
// @Param		offset		query		int		false	"Offset"
// @Success	200	{array}		[]object
// @Failure	401	{object}	map[string]string
// @Router		/api/v1/users [get]
func (h *DiscoveryHandler) Search(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	f := repository.DiscoveryFilters{ExcludeID: id, Limit: 500}
	if blockedIDs, err := h.blockRepo.ListBlockedIDs(c.Request.Context(), id); err == nil {
		f.ExcludeIDs = blockedIDs
	}

	me, err := h.profileRepo.GetByUserID(c.Request.Context(), id)
	if err != nil || me == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "complete your profile before using discovery"})
		return
	}
	if !isDiscoveryProfileReady(me) {
		c.JSON(http.StatusForbidden, gin.H{"error": "complete your profile before using discovery"})
		return
	}
	if len(me.SexualPreference) > 0 {
		f.Genders = me.SexualPreference
	}
	if me.Gender != nil && *me.Gender != "" {
		f.ReciprocityUserGender = *me.Gender
	}
	if me.Latitude != nil && me.Longitude != nil {
		f.UserLat = me.Latitude
		f.UserLon = me.Longitude
	}
	if me.City != nil {
		f.PreferredCity = *me.City
	}
	if myTags, err := h.profileRepo.GetTags(c.Request.Context(), id); err == nil {
		f.Tags = myTags
	}
	if v := c.Query("gender"); v != "" {
		f.Genders = parseCommaList(v)
	}
	if v := c.Query("interest"); v != "" {
		f.Genders = parseCommaList(v)
	}
	if v := c.Query("min_age"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinAge = n
		}
	}
	if v := c.Query("max_age"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MaxAge = n
		}
	}
	if v := c.Query("min_fame"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MinFame = n
		}
	}
	if v := c.Query("max_fame"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.MaxFame = n
		}
	}
	if v := strings.TrimSpace(c.Query("city")); v != "" {
		f.City = v
	}
	if v := strings.TrimSpace(c.Query("tags")); v != "" {
		f.Tags = nil
		f.StrictTags = true
		raw := strings.Split(v, ",")
		for _, t := range raw {
			t = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(t)), "#")
			if t != "" {
				f.Tags = append(f.Tags, t)
			}
		}
	}
	if v := c.Query("max_distance_km"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.MaxDistanceKm = n
		}
	}
	if v := c.Query("current_lat"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= -90 && n <= 90 {
			f.UserLat = &n
		}
	}
	if v := c.Query("current_lon"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= -180 && n <= 180 {
			f.UserLon = &n
		}
	}
	if v := c.Query("relationship_goal"); v != "" {
		f.RelationshipGoals = parseCommaList(v)
	}
	if v := c.Query("sort_by"); v != "" {
		f.SortBy = v
	}
	if v := c.Query("sort_order"); v != "" {
		f.SortOrder = v
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			f.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Offset = n
		}
	}
	applyDefaultDiscoveryGenders(&f)

	cards, err := h.discoveryRepo.Search(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(cards))
	for i, card := range cards {
		item := toUserCardResp(&card)
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), card.ID); err == nil && p != nil {
			item["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
		result[i] = item
	}
	c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary	Get user public profile
// @Tags		discovery
// @Security	BearerAuth
// @Produce	json
// @Param		id	path		string	true	"User ID"
// @Success	200	{object}	object
// @Failure	401	{object}	map[string]string
// @Failure	404	{object}	map[string]string
// @Router		/api/v1/users/{id} [get]
func (h *DiscoveryHandler) GetByID(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	viewerID := userID.(uuid.UUID)

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	u, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	p, _ := h.profileRepo.GetByUserID(c.Request.Context(), id)
	photos, _ := h.photoRepo.ListByUser(c.Request.Context(), id)

	resp := gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
	}
	if p != nil {
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
		resp["fame_rating"] = p.FameRating
	}
	if tags, err := h.profileRepo.GetTags(c.Request.Context(), id); err == nil {
		resp["tags"] = tags
	}
	if len(photos) > 0 {
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

	if viewerID != id {
		isBlocked, _ := h.blockRepo.IsBlockedEither(c.Request.Context(), viewerID, id)
		if isBlocked {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		_ = h.profileRepo.AddProfileView(c.Request.Context(), viewerID, id)
		if blocked, _ := h.blockRepo.BlockedBy(c.Request.Context(), id, viewerID); !blocked {
			notif, _ := h.notifRepo.Create(c.Request.Context(), id, &viewerID, "visit", nil, "Someone visited your profile")
			pushNotification(h.hub, id, notif)
		}
		if _, err := h.profileRepo.RecalculateFameRating(c.Request.Context(), id); err == nil {
			_ = h.syncSvc.SyncUser(c.Request.Context(), id)
		}
	}
	if viewerID != id {
		if likedMe, err := h.likeRepo.Exists(c.Request.Context(), id, viewerID); err == nil {
			resp["liked_me"] = likedMe
		}
		if iLiked, err := h.likeRepo.Exists(c.Request.Context(), viewerID, id); err == nil {
			resp["i_liked"] = iLiked
		}
		if isMatch, err := h.likeRepo.IsMatch(c.Request.Context(), viewerID, id); err == nil {
			resp["is_match"] = isMatch
		}
	}

	c.JSON(http.StatusOK, resp)
}

func toUserCardResp(c *repository.UserCard) gin.H {
	resp := gin.H{
		"id":          c.ID,
		"username":    c.Username,
		"first_name":  c.FirstName,
		"last_name":   c.LastName,
		"fame_rating": c.FameRating,
	}
	if c.Gender != nil {
		resp["gender"] = *c.Gender
	}
	if len(c.SexualPreference) > 0 {
		resp["sexual_preference"] = c.SexualPreference
	}
	if c.RelationshipGoal != nil {
		resp["relationship_goal"] = *c.RelationshipGoal
	}
	if c.BirthDate != nil {
		resp["birth_date"] = c.BirthDate.Format("2006-01-02")
	}
	if c.Bio != nil {
		resp["bio"] = *c.Bio
	}
	if c.City != nil {
		resp["city"] = *c.City
	}
	if len(c.Tags) > 0 {
		resp["tags"] = c.Tags
	}
	if c.Latitude != nil {
		resp["latitude"] = *c.Latitude
	}
	if c.Longitude != nil {
		resp["longitude"] = *c.Longitude
	}
	return resp
}

func parseCommaList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(strings.ToLower(v))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func isDiscoveryProfileReady(me *repository.Profile) bool {
	if me == nil {
		return false
	}
	if me.Gender == nil || strings.TrimSpace(*me.Gender) == "" {
		return false
	}
	if len(me.SexualPreference) == 0 {
		return false
	}
	if me.BirthDate == nil {
		return false
	}
	if me.City == nil || strings.TrimSpace(*me.City) == "" {
		return false
	}
	return true
}

func applyDefaultDiscoveryGenders(f *repository.DiscoveryFilters) {
	if f == nil {
		return
	}
	if len(f.Genders) == 0 {
		f.Genders = []string{"female", "male"}
	}
}
