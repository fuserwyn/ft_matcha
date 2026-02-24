package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/repository"
	"matcha/api/internal/websocket"
)

type PresenceHandler struct {
	presence *repository.PresenceRepository
	hub      *websocket.Hub
}

func NewPresenceHandler(presence *repository.PresenceRepository, hub *websocket.Hub) *PresenceHandler {
	return &PresenceHandler{presence: presence, hub: hub}
}

// Get godoc
// @Summary	Get user presence
// @Tags		presence
// @Security	BearerAuth
// @Param		id	path		string	true	"User ID"
// @Success	200	{object}	object
// @Failure	400	{object}	map[string]string
// @Router		/api/v1/presence/{id} [get]
func (h *PresenceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	lastSeen, err := h.presence.GetLastSeen(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   id,
		"is_online": h.hub.IsOnline(id),
		"last_seen": lastSeen,
	})
}
