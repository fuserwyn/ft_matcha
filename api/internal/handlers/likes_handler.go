package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type LikesHandler struct {
	likeRepo *repository.LikeRepository
	userRepo *repository.UserRepository
}

func NewLikesHandler(likeRepo *repository.LikeRepository, userRepo *repository.UserRepository) *LikesHandler {
	return &LikesHandler{likeRepo: likeRepo, userRepo: userRepo}
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

	isMatch, _ := h.likeRepo.IsMatch(c.Request.Context(), myID, likedID)
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
	for i := range cards {
		result[i] = toUserCardResp(&cards[i])
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
	for i := range cards {
		result[i] = toUserCardResp(&cards[i])
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
	for i := range cards {
		result[i] = toUserCardResp(&cards[i])
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
