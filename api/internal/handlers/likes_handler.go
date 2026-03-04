package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
	"matcha/api/internal/storage"
	ws "matcha/api/internal/websocket"
)

type LikesHandler struct {
	likeRepo         *repository.LikeRepository
	userRepo         *repository.UserRepository
	profileRepo      *repository.ProfileRepository
	photoRepo        *repository.PhotoRepository
	blockRepo        *repository.BlockRepository
	notificationRepo *repository.NotificationRepository
	mailer           *services.Mailer
	syncSvc          *services.SyncService
	hub              *ws.Hub
	photoStore       *storage.MinIO
	apiBaseURL       string
}

func NewLikesHandler(
	likeRepo *repository.LikeRepository,
	userRepo *repository.UserRepository,
	profileRepo *repository.ProfileRepository,
	photoRepo *repository.PhotoRepository,
	blockRepo *repository.BlockRepository,
	notificationRepo *repository.NotificationRepository,
	mailer *services.Mailer,
	syncSvc *services.SyncService,
	hub *ws.Hub,
	photoStore *storage.MinIO,
	apiBaseURL string,
) *LikesHandler {
	return &LikesHandler{
		likeRepo:         likeRepo,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		photoRepo:        photoRepo,
		blockRepo:        blockRepo,
		notificationRepo: notificationRepo,
		mailer:           mailer,
		syncSvc:          syncSvc,
		hub:              hub,
		photoStore:       photoStore,
		apiBaseURL:       apiBaseURL,
	}
}

// Like godoc
// @Summary	Like a user
// @Tags		likes
// @Security	BearerAuth
// @Param		id	path		string	true	"User ID to like"
// @Success	201	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Failure	409	{object}	map[string]string
// @Failure	404	{object}	map[string]string
// @Router		/api/v1/users/{id}/like [post]
func (h *LikesHandler) Like(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	myID := userID.(uuid.UUID)

	likedIDStr := c.Param("id")
	likedID, err := uuid.Parse(likedIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if myID == likedID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot like yourself"})
		return
	}
	if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), myID); err != nil || p == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "set profile picture before liking users"})
		return
	}
	isBlocked, err := h.blockRepo.IsBlockedEither(c.Request.Context(), myID, likedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isBlocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot like blocked user"})
		return
	}

	exists, err := h.userRepo.GetByID(c.Request.Context(), likedID)
	if err != nil || exists == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	alreadyLiked, err := h.likeRepo.Exists(c.Request.Context(), myID, likedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if alreadyLiked {
		c.JSON(http.StatusConflict, gin.H{"error": "already liked"})
		return
	}

	if err := h.likeRepo.Create(c.Request.Context(), myID, likedID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.profileRepo.RecalculateFameRating(c.Request.Context(), likedID); err == nil {
		_ = h.syncSvc.SyncUser(c.Request.Context(), likedID)
	}
	notif, _ := h.notificationRepo.Create(c.Request.Context(), likedID, &myID, "like", nil, "You have a new like")
	pushNotification(h.hub, likedID, notif)
	actor, _ := h.userRepo.GetByID(c.Request.Context(), myID)
	if actor != nil {
		_ = h.mailer.Send(
			exists.Email,
			"New like on Matcha",
			actor.FirstName+" "+actor.LastName+" liked your profile.",
		)
	}

	isMatch, _ := h.likeRepo.IsMatch(c.Request.Context(), myID, likedID)
	if isMatch {
		n1, _ := h.notificationRepo.Create(c.Request.Context(), likedID, &myID, "match", nil, "It's a match")
		n2, _ := h.notificationRepo.Create(c.Request.Context(), myID, &likedID, "match", nil, "It's a match")
		pushNotification(h.hub, likedID, n1)
		pushNotification(h.hub, myID, n2)
		_ = h.mailer.Send(exists.Email, "It's a match on Matcha", "You have a new match.")
		if actor != nil {
			_ = h.mailer.Send(actor.Email, "It's a match on Matcha", "You have a new match.")
		}
	}
	c.JSON(http.StatusCreated, gin.H{
		"liked_user_id": likedID,
		"is_match":      isMatch,
	})
}

// Unlike godoc
// @Summary	Remove like
// @Tags		likes
// @Security	BearerAuth
// @Param		id	path		string	true	"User ID to unlike"
// @Success	204	""
// @Failure	400	{object}	map[string]string
// @Router		/api/v1/users/{id}/like [delete]
func (h *LikesHandler) Unlike(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	myID := userID.(uuid.UUID)

	likedIDStr := c.Param("id")
	likedID, err := uuid.Parse(likedIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	_ = h.likeRepo.Delete(c.Request.Context(), myID, likedID)
	isBlocked, _ := h.blockRepo.IsBlockedEither(c.Request.Context(), myID, likedID)
	if !isBlocked {
		n, _ := h.notificationRepo.Create(c.Request.Context(), likedID, &myID, "unlike", nil, "A user unliked you")
		pushNotification(h.hub, likedID, n)
	}
	if _, err := h.profileRepo.RecalculateFameRating(c.Request.Context(), likedID); err == nil {
		_ = h.syncSvc.SyncUser(c.Request.Context(), likedID)
	}
	c.Status(http.StatusNoContent)
}

// GetLikedByMe godoc
// @Summary	Get users I liked
// @Tags		likes
// @Security	BearerAuth
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200	{array}		object
// @Router		/api/v1/likes [get]
func (h *LikesHandler) GetLikedByMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit, offset := parseLimitOffset(c)

	cards, err := h.likeRepo.GetLikedByMe(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(cards))
	for i, card := range cards {
		item := toUserCardResp(&cards[i])
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), card.ID); err == nil && p != nil {
			item["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
		result[i] = item
	}
	c.JSON(http.StatusOK, result)
}

// GetLikedMe godoc
// @Summary	Get users who liked me
// @Tags		likes
// @Security	BearerAuth
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200	{array}		object
// @Router		/api/v1/likes/me [get]
func (h *LikesHandler) GetLikedMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit, offset := parseLimitOffset(c)

	cards, err := h.likeRepo.GetLikedMe(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(cards))
	for i, card := range cards {
		item := toUserCardResp(&cards[i])
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), card.ID); err == nil && p != nil {
			item["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
		result[i] = item
	}
	c.JSON(http.StatusOK, result)
}

// GetMatches godoc
// @Summary	Get mutual matches
// @Tags		likes
// @Security	BearerAuth
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		offset	query		int	false	"Offset"
// @Success	200	{array}		object
// @Router		/api/v1/matches [get]
func (h *LikesHandler) GetMatches(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit, offset := parseLimitOffset(c)

	cards, err := h.likeRepo.GetMatches(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(cards))
	for i, card := range cards {
		item := toUserCardResp(&cards[i])
		if p, err := h.photoRepo.GetPrimaryByUser(c.Request.Context(), card.ID); err == nil && p != nil {
			item["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
		result[i] = item
	}
	c.JSON(http.StatusOK, result)
}

func parseLimitOffset(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func pushNotification(hub *ws.Hub, userID uuid.UUID, n *repository.Notification) {
	if hub == nil || n == nil {
		return
	}
	hub.SendToUser(userID, gin.H{
		"type": "notification",
		"data": gin.H{
			"id":         n.ID,
			"user_id":    n.UserID,
			"actor_id":   n.ActorID,
			"type":       n.Type,
			"entity_id":  n.EntityID,
			"content":    n.Content,
			"is_read":    n.IsRead,
			"created_at": n.CreatedAt,
			"read_at":    n.ReadAt,
		},
	})
}
