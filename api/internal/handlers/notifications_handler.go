package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type NotificationsHandler struct {
	notifications *repository.NotificationRepository
}

func NewNotificationsHandler(notifications *repository.NotificationRepository) *NotificationsHandler {
	return &NotificationsHandler{notifications: notifications}
}

// List godoc
// @Summary	List my notifications
// @Tags		notifications
// @Security	BearerAuth
// @Param		unread_only	query		bool	false	"Only unread notifications"
// @Param		limit		query		int		false	"Limit (default 20)"
// @Param		offset		query		int		false	"Offset"
// @Success	200			{array}		object
// @Router		/api/v1/notifications [get]
func (h *NotificationsHandler) List(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	limit, offset := parseLimitOffset(c)
	unreadOnly, _ := strconv.ParseBool(c.DefaultQuery("unread_only", "false"))

	items, err := h.notifications.ListByUser(c.Request.Context(), id, unreadOnly, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]gin.H, len(items))
	for i, n := range items {
		resp[i] = gin.H{
			"id":         n.ID,
			"user_id":    n.UserID,
			"actor_id":   n.ActorID,
			"type":       n.Type,
			"entity_id":  n.EntityID,
			"content":    n.Content,
			"is_read":    n.IsRead,
			"created_at": n.CreatedAt,
			"read_at":    n.ReadAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// MarkAllRead godoc
// @Summary	Mark all notifications as read
// @Tags		notifications
// @Security	BearerAuth
// @Success	200	{object}	object
// @Router		/api/v1/notifications/read-all [patch]
func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)

	affected, err := h.notifications.MarkAllRead(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": affected})
}
