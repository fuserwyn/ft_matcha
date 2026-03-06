package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
	"matcha/api/internal/storage"
	ws "matcha/api/internal/websocket"
)

type ChatHandler struct {
	messageRepo      *repository.MessageRepository
	likeRepo         *repository.LikeRepository
	userRepo         *repository.UserRepository
	blockRepo        *repository.BlockRepository
	notificationRepo *repository.NotificationRepository
	mailer           *services.Mailer
	hub              *ws.Hub
	store            *storage.MinIO
}

func NewChatHandler(
	messageRepo *repository.MessageRepository,
	likeRepo *repository.LikeRepository,
	userRepo *repository.UserRepository,
	blockRepo *repository.BlockRepository,
	notificationRepo *repository.NotificationRepository,
	mailer *services.Mailer,
	hub *ws.Hub,
	store *storage.MinIO,
) *ChatHandler {
	return &ChatHandler{
		messageRepo:      messageRepo,
		likeRepo:         likeRepo,
		userRepo:         userRepo,
		blockRepo:        blockRepo,
		notificationRepo: notificationRepo,
		mailer:           mailer,
		hub:              hub,
		store:            store,
	}
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

	otherID, err := h.validateChatPeer(c, myID)
	if err != nil {
		if strings.Contains(err.Error(), "match") || strings.Contains(err.Error(), "blocked") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	var notif *repository.Notification
	if blocked, _ := h.blockRepo.BlockedBy(c.Request.Context(), otherID, myID); !blocked {
		notif, _ = h.notificationRepo.Create(
			c.Request.Context(),
			otherID,
			&myID,
			"message",
			&m.ID,
			"New message from match",
		)
	}
	fromUser, _ := h.userRepo.GetByID(c.Request.Context(), myID)
	toUser, _ := h.userRepo.GetByID(c.Request.Context(), otherID)
	if fromUser != nil && toUser != nil && notif != nil {
		_ = h.mailer.Send(
			toUser.Email,
			"New message on Matcha",
			fromUser.FirstName+" "+fromUser.LastName+" sent you a message.",
		)
	}
	if h.hub != nil {
		event := gin.H{
			"type": "message",
			"data": gin.H{
				"id":           m.ID,
				"sender_id":    m.SenderID,
				"receiver_id":  m.ReceiverID,
				"content":      m.Content,
				"message_type": m.MessageType,
				"media_url":    m.MediaURL,
				"created_at":   m.CreatedAt,
				"is_read":      m.IsRead,
				"read_at":      m.ReadAt,
			},
		}
		h.hub.SendToUser(myID, event)
		h.hub.SendToUser(otherID, event)
		if notif != nil {
			h.hub.SendToUser(otherID, gin.H{
				"type": "notification",
				"data": gin.H{
					"id":         notif.ID,
					"user_id":    notif.UserID,
					"actor_id":   notif.ActorID,
					"type":       notif.Type,
					"entity_id":  notif.EntityID,
					"content":    notif.Content,
					"is_read":    notif.IsRead,
					"created_at": notif.CreatedAt,
					"read_at":    notif.ReadAt,
				},
			})
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           m.ID,
		"sender_id":    m.SenderID,
		"receiver_id":  m.ReceiverID,
		"content":      m.Content,
		"message_type": m.MessageType,
		"media_url":    m.MediaURL,
		"created_at":   m.CreatedAt,
		"is_read":      m.IsRead,
		"read_at":      m.ReadAt,
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
	isBlocked, err := h.blockRepo.IsBlockedEither(c.Request.Context(), myID, otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isBlocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot view messages with blocked user"})
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
			"id":           m.ID,
			"sender_id":    m.SenderID,
			"receiver_id":  m.ReceiverID,
			"content":      m.Content,
			"message_type": m.MessageType,
			"media_url":    m.MediaURL,
			"created_at":   m.CreatedAt,
			"is_read":      m.IsRead,
			"read_at":      m.ReadAt,
		}
	}
	c.JSON(http.StatusOK, result)
}

var allowedVoiceContentTypes = map[string]struct{}{
	"audio/webm":      {},
	"video/webm":      {},
	"audio/mp4":       {},
	"audio/mpeg":      {},
	"audio/ogg":       {},
	"application/ogg": {},
	"audio/wav":       {},
	"audio/x-wav":     {},
}

var allowedVoiceExtensions = map[string]struct{}{
	".webm": {},
	".m4a":  {},
	".mp3":  {},
	".ogg":  {},
	".wav":  {},
}

const maxVoiceBytes = 10 * 1024 * 1024

func normalizeContentType(v string) string {
	contentType := strings.ToLower(strings.TrimSpace(v))
	if i := strings.Index(contentType, ";"); i > 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	return contentType
}

func voiceContentTypeByExt(ext string) string {
	switch ext {
	case ".webm":
		return "audio/webm"
	case ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}

func (h *ChatHandler) SendVoiceMessage(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	myID := userID.(uuid.UUID)

	otherID, err := h.validateChatPeer(c, myID)
	if err != nil {
		if strings.Contains(err.Error(), "match") || strings.Contains(err.Error(), "blocked") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "voice storage is not configured"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if _, ok := allowedVoiceExtensions[ext]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported voice file extension"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	buf, err := io.ReadAll(io.LimitReader(src, maxVoiceBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read uploaded file"})
		return
	}
	if len(buf) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voice file is empty"})
		return
	}
	if len(buf) > maxVoiceBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voice file is too large (max 10MB)"})
		return
	}

	detectedType := normalizeContentType(http.DetectContentType(buf))
	headerType := normalizeContentType(fileHeader.Header.Get("Content-Type"))
	contentType := ""
	if _, ok := allowedVoiceContentTypes[headerType]; ok {
		contentType = headerType
	} else if _, ok := allowedVoiceContentTypes[detectedType]; ok {
		contentType = detectedType
	} else {
		contentType = voiceContentTypeByExt(ext)
	}
	if contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported voice content type"})
		return
	}

	voiceID := uuid.NewString()
	objectKey := fmt.Sprintf("voice/%s/%s%s", myID.String(), voiceID, ext)
	mediaURL, err := h.store.PutObject(c.Request.Context(), objectKey, bytes.NewReader(buf), int64(len(buf)), contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store voice file"})
		return
	}

	voiceLabel := "Voice message"
	m, err := h.messageRepo.CreateWithMeta(c.Request.Context(), myID, otherID, voiceLabel, "voice", &mediaURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.hub != nil {
		event := gin.H{
			"type": "message",
			"data": gin.H{
				"id":           m.ID,
				"sender_id":    m.SenderID,
				"receiver_id":  m.ReceiverID,
				"content":      m.Content,
				"message_type": m.MessageType,
				"media_url":    m.MediaURL,
				"created_at":   m.CreatedAt,
				"is_read":      m.IsRead,
				"read_at":      m.ReadAt,
			},
		}
		h.hub.SendToUser(myID, event)
		h.hub.SendToUser(otherID, event)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           m.ID,
		"sender_id":    m.SenderID,
		"receiver_id":  m.ReceiverID,
		"content":      m.Content,
		"message_type": m.MessageType,
		"media_url":    m.MediaURL,
		"created_at":   m.CreatedAt,
		"is_read":      m.IsRead,
		"read_at":      m.ReadAt,
	})
}

// MarkRead godoc
// @Summary	Mark messages from user as read
// @Tags		chat
// @Security	BearerAuth
// @Param		id	path		string	true	"Sender user ID"
// @Success	200	{object}	object
// @Failure	400	{object}	map[string]string
// @Failure	403	{object}	map[string]string
// @Router		/api/v1/users/{id}/messages/read [patch]
func (h *ChatHandler) MarkRead(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "can only mark messages with matches"})
		return
	}
	isBlocked, err := h.blockRepo.IsBlockedEither(c.Request.Context(), myID, otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isBlocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot mark messages with blocked user"})
		return
	}

	affected, err := h.messageRepo.MarkReadFromSender(c.Request.Context(), myID, otherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if affected > 0 && h.hub != nil {
		event := gin.H{
			"type": "message_read",
			"data": gin.H{
				"sender_id": otherID,
				"reader_id": myID,
				"read_at":   time.Now().UTC(),
			},
		}
		h.hub.SendToUser(otherID, event)
		h.hub.SendToUser(myID, event)
	}
	c.JSON(http.StatusOK, gin.H{"updated": affected})
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

func (h *ChatHandler) validateChatPeer(c *gin.Context, myID uuid.UUID) (uuid.UUID, error) {
	otherID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid id")
	}
	if myID == otherID {
		return uuid.Nil, fmt.Errorf("cannot message yourself")
	}
	isMatch, err := h.likeRepo.IsMatch(c.Request.Context(), myID, otherID)
	if err != nil {
		return uuid.Nil, err
	}
	if !isMatch {
		return uuid.Nil, fmt.Errorf("can only message matches")
	}
	isBlocked, err := h.blockRepo.IsBlockedEither(c.Request.Context(), myID, otherID)
	if err != nil {
		return uuid.Nil, err
	}
	if isBlocked {
		return uuid.Nil, fmt.Errorf("cannot message blocked user")
	}
	return otherID, nil
}
