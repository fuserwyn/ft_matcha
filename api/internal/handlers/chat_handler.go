package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type ChatHandler struct {
	messageRepo *repository.MessageRepository
	likeRepo    *repository.LikeRepository
}

func NewChatHandler(messageRepo *repository.MessageRepository, likeRepo *repository.LikeRepository) *ChatHandler {
	return &ChatHandler{messageRepo: messageRepo, likeRepo: likeRepo}
}

type SendMessageReq struct {
	Content string `json:"content" binding:"required"`
}

// SendMessage godoc
// @Summary	Send message to a match
// @Tags		chat
// @Security	BearerAuth
// @Accept		json
// @Produce	json
// @Param		id		path		string			true	"User ID (must be a match)"
// @Param		body	body		SendMessageReq	true	"Message content"
// @Success	201		{object}	object
// @Failure	400		{object}	map[string]string
// @Failure	403		{object}	map[string]string
// @Failure	404		{object}	map[string]string
// @Router		/api/v1/users/{id}/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	myID := userID.(uuid.UUID)

	otherIDStr := c.Param("id")
	otherID, err := uuid.Parse(otherIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if myID == otherID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot message yourself"})
		return
	}

	isMatch, err := h.likeRepo.IsMatch(c.Request.Context(), myID, otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMatch {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only message matches"})
		return
	}

	var req SendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Content) == 0 || len(req.Content) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content must be 1-2000 characters"})
		return
	}

	m, err := h.messageRepo.Create(c.Request.Context(), myID, otherID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         m.ID,
		"sender_id":  m.SenderID,
		"receiver_id": m.ReceiverID,
		"content":    m.Content,
		"created_at": m.CreatedAt,
	})
}

// GetMessages godoc
// @Summary	Get messages with a match
// @Tags		chat
// @Security	BearerAuth
// @Param		id		path		string	true	"User ID (must be a match)"
// @Param		limit	query		int		false	"Limit (default 50)"
// @Param		offset	query		int		false	"Offset"
// @Success	200		{array}		object
// @Failure	400		{object}	map[string]string
// @Failure	403		{object}	map[string]string
// @Router		/api/v1/users/{id}/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	myID := userID.(uuid.UUID)

	otherIDStr := c.Param("id")
	otherID, err := uuid.Parse(otherIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if myID == otherID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	isMatch, err := h.likeRepo.IsMatch(c.Request.Context(), myID, otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !isMatch {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only view messages with matches"})
		return
	}

	limit, offset := parseLimitOffsetChat(c)

	msgs, err := h.messageRepo.GetBetween(c.Request.Context(), myID, otherID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]gin.H, len(msgs))
	for i, m := range msgs {
		result[i] = gin.H{
			"id":          m.ID,
			"sender_id":   m.SenderID,
			"receiver_id": m.ReceiverID,
			"content":     m.Content,
			"created_at":  m.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, result)
}

func parseLimitOffsetChat(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0
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
	return limit, offset
}
