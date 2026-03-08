package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type BlocksHandler struct {
	blocks     *repository.BlockRepository
	users      *repository.UserRepository
	profiles   *repository.ProfileRepository
	photos     *repository.PhotoRepository
	apiBaseURL string
}

func NewBlocksHandler(
	blocks *repository.BlockRepository,
	users *repository.UserRepository,
	profiles *repository.ProfileRepository,
	photos *repository.PhotoRepository,
	apiBaseURL string,
) *BlocksHandler {
	return &BlocksHandler{
		blocks:     blocks,
		users:      users,
		profiles:   profiles,
		photos:     photos,
		apiBaseURL: strings.TrimRight(apiBaseURL, "/"),
	}
}

// BlockUser godoc
// @Summary	Block a user
// @Tags		blocks
// @Security	BearerAuth
// @Produce	json
// @Param		id	path		string	true	"User ID to block"
// @Success	200	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Failure	404	{object}	map[string]string
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/users/{id}/block [post]
func (h *BlocksHandler) BlockUser(c *gin.Context) {
	me := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	otherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if me == otherID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot block yourself"})
		return
	}
	if u, err := h.users.GetByID(c.Request.Context(), otherID); err != nil || u == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := h.blocks.Block(c.Request.Context(), me, otherID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UnblockUser godoc
// @Summary	Unblock a user
// @Tags		blocks
// @Security	BearerAuth
// @Param		id	path		string	true	"User ID to unblock"
// @Success	204	"No Content"
// @Failure	400	{object}	map[string]string
// @Failure	500	{object}	map[string]string
// @Router		/api/v1/users/{id}/block [delete]
func (h *BlocksHandler) UnblockUser(c *gin.Context) {
	me := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	otherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.blocks.Unblock(c.Request.Context(), me, otherID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ListBlockedUsers godoc
// @Summary	List users blocked by me
// @Tags		blocks
// @Security	BearerAuth
// @Param		limit	query		int	false	"Limit (default 20)"
// @Param		cursor	query		string	false	"Cursor from previous response"
// @Success	200		{object}	object
// @Failure	500		{object}	map[string]string
// @Router		/api/v1/blocks [get]
func (h *BlocksHandler) ListBlockedUsers(c *gin.Context) {
	me := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	limit := parseCursorLimit(c, 20, 50)
	cursor, err := parsePageCursor(c.Query("cursor"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
		return
	}

	rows, err := h.blocks.ListBlockedByMeCursor(c.Request.Context(), me, limit+1, cursorTime(cursor), cursorID(cursor))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	result := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		id := row.BlockedUserID
		u, err := h.users.GetByID(c.Request.Context(), id)
		if err != nil || u == nil {
			continue
		}
		item := gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
		}
		if p, err := h.profiles.GetByUserID(c.Request.Context(), id); err == nil && p != nil {
			item["fame_rating"] = p.FameRating
			if p.BirthDate != nil {
				item["birth_date"] = p.BirthDate.Format("2006-01-02")
			}
			if p.Gender != nil {
				item["gender"] = *p.Gender
			}
			if p.Bio != nil {
				item["bio"] = *p.Bio
			}
			if p.City != nil {
				item["city"] = *p.City
			}
		}
		if tags, err := h.profiles.GetTags(c.Request.Context(), id); err == nil && len(tags) > 0 {
			item["tags"] = tags
		}
		if p, err := h.photos.GetPrimaryByUser(c.Request.Context(), id); err == nil && p != nil {
			item["primary_photo_url"] = photoURL(p, h.apiBaseURL)
		}
		result = append(result, item)
	}

	nextCursor := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = encodePageCursor(last.CursorTime, last.CursorID)
	}
	c.JSON(http.StatusOK, gin.H{
		"items":       result,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}
