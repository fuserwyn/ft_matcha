package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
)

type BlocksHandler struct {
	blocks *repository.BlockRepository
	users  *repository.UserRepository
}

func NewBlocksHandler(blocks *repository.BlockRepository, users *repository.UserRepository) *BlocksHandler {
	return &BlocksHandler{blocks: blocks, users: users}
}

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
