package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/repository"
)

func TouchPresence(presenceRepo *repository.PresenceRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDRaw, ok := c.Get(UserIDKey)
		if ok {
			if userID, castOK := userIDRaw.(uuid.UUID); castOK {
				_ = presenceRepo.UpsertLastSeen(c.Request.Context(), userID, time.Now().UTC())
			}
		}
		c.Next()
	}
}
