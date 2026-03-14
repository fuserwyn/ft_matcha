package websocket

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
)

type wsClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type ChatHandler struct {
	hub              *Hub
	likeRepo         *repository.LikeRepository
	messageRepo      *repository.MessageRepository
	userRepo         *repository.UserRepository
	blockRepo        *repository.BlockRepository
	notificationRepo *repository.NotificationRepository
	presenceRepo     *repository.PresenceRepository
	mailer           *services.Mailer
	jwtSecret        string
	upgrader         gws.Upgrader

	rateMu   sync.Mutex
	rateByID map[uuid.UUID]rateState
}

type rateState struct {
	windowStart time.Time
	count       int
}

type incomingMessage struct {
	Type      string `json:"type"`
	ToUserID  string `json:"to_user_id"`
	Content   string `json:"content"`
	CallID    string `json:"call_id"`
	Mode      string `json:"mode"`
	SDP       string `json:"sdp"`
	Candidate any    `json:"candidate"`
}

func NewChatHandler(
	hub *Hub,
	likeRepo *repository.LikeRepository,
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	blockRepo *repository.BlockRepository,
	notificationRepo *repository.NotificationRepository,
	presenceRepo *repository.PresenceRepository,
	mailer *services.Mailer,
	jwtSecret string,
) *ChatHandler {
	return &ChatHandler{
		hub:              hub,
		likeRepo:         likeRepo,
		messageRepo:      messageRepo,
		userRepo:         userRepo,
		blockRepo:        blockRepo,
		notificationRepo: notificationRepo,
		presenceRepo:     presenceRepo,
		mailer:           mailer,
		jwtSecret:        jwtSecret,
		rateByID:         make(map[uuid.UUID]rateState),
		upgrader: gws.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

func (h *ChatHandler) Handle(c *gin.Context) {
	userID, err := h.authenticate(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := h.hub.Register(userID, conn)
	defer h.hub.Unregister(userID, client)
	defer func() {
		_ = h.presenceRepo.UpsertLastSeen(context.Background(), userID, time.Now().UTC())
	}()

	h.hub.SendToUser(userID, gin.H{
		"type": "connected",
		"data": gin.H{"user_id": userID},
	})

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	stopPing := make(chan struct{})
	defer close(stopPing)
	go h.pingLoop(userID, client, stopPing)

	for {
		var in incomingMessage
		if err := conn.ReadJSON(&in); err != nil {
			if gws.IsUnexpectedCloseError(err, gws.CloseGoingAway, gws.CloseAbnormalClosure) {
				h.hub.SendToUser(userID, gin.H{"type": "error", "error": "connection closed"})
			}
			break
		}

		if err := h.processIncoming(userID, in); err != nil {
			h.hub.SendToUser(userID, gin.H{"type": "error", "error": err.Error()})
		}
	}
}

func (h *ChatHandler) processIncoming(fromUserID uuid.UUID, in incomingMessage) error {
	kind := strings.ToLower(strings.TrimSpace(in.Type))
	if kind == "" || kind == "message" {
		return h.processChatMessage(fromUserID, in)
	}
	return h.processCallSignal(fromUserID, in, kind)
}

func (h *ChatHandler) processChatMessage(fromUserID uuid.UUID, in incomingMessage) error {
	if !h.allowMessage(fromUserID) {
		return errors.New("rate limit exceeded: too many messages")
	}
	toUserID, err := h.validateMatchAndBlock(fromUserID, in.ToUserID)
	if err != nil {
		return err
	}
	content := strings.TrimSpace(in.Content)
	if len(content) == 0 || len(content) > 2000 {
		return errors.New("content must be 1-2000 characters")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg, err := h.messageRepo.Create(ctx, fromUserID, toUserID, content)
	if err != nil {
		return err
	}
	notif, _ := h.notificationRepo.Create(
		ctx,
		toUserID,
		&fromUserID,
		"message",
		&msg.ID,
		"New message from match",
	)
	fromUser, _ := h.userRepo.GetByID(ctx, fromUserID)
	toUser, _ := h.userRepo.GetByID(ctx, toUserID)
	if fromUser != nil && toUser != nil {
		_ = h.mailer.Send(
			toUser.Email,
			"New message on Matcha",
			fromUser.FirstName+" "+fromUser.LastName+" sent you a message.",
		)
	}

	event := gin.H{
		"type": "message",
		"data": gin.H{
			"id":           msg.ID,
			"sender_id":    msg.SenderID,
			"receiver_id":  msg.ReceiverID,
			"content":      msg.Content,
			"message_type": msg.MessageType,
			"media_url":    msg.MediaURL,
			"created_at":   msg.CreatedAt,
			"is_read":      msg.IsRead,
			"read_at":      msg.ReadAt,
		},
	}
	h.hub.SendToUser(fromUserID, event)
	h.hub.SendToUser(toUserID, event)
	if notif != nil {
		h.hub.SendToUser(toUserID, gin.H{
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
	return nil
}

func (h *ChatHandler) processCallSignal(fromUserID uuid.UUID, in incomingMessage, kind string) error {
	toUserID, err := h.validateMatchAndBlock(fromUserID, in.ToUserID)
	if err != nil {
		return err
	}
	callID := strings.TrimSpace(in.CallID)
	if callID == "" {
		return errors.New("call_id is required")
	}
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = "video"
	}
	if mode != "video" && mode != "audio" {
		return errors.New("mode must be video or audio")
	}

	switch kind {
	case "call_invite", "call_accept":
		if strings.TrimSpace(in.SDP) == "" {
			return errors.New("sdp is required")
		}
	case "call_ice":
		if in.Candidate == nil {
			return errors.New("candidate is required")
		}
	case "call_reject", "call_end":
	default:
		return errors.New("unsupported event type")
	}

	event := gin.H{
		"type": kind,
		"data": gin.H{
			"call_id":      callID,
			"from_user_id": fromUserID,
			"to_user_id":   toUserID,
			"mode":         mode,
			"sdp":          in.SDP,
			"candidate":    in.Candidate,
		},
	}
	h.hub.SendToUser(fromUserID, event)
	h.hub.SendToUser(toUserID, event)
	return nil
}

func (h *ChatHandler) validateMatchAndBlock(fromUserID uuid.UUID, toUserIDRaw string) (uuid.UUID, error) {
	toUserID, err := uuid.Parse(strings.TrimSpace(toUserIDRaw))
	if err != nil {
		return uuid.Nil, errors.New("invalid to_user_id")
	}
	if toUserID == fromUserID {
		return uuid.Nil, errors.New("cannot message yourself")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	isMatch, err := h.likeRepo.IsMatch(ctx, fromUserID, toUserID)
	if err != nil {
		return uuid.Nil, err
	}
	if !isMatch {
		return uuid.Nil, errors.New("can only message matches")
	}
	isBlocked, err := h.blockRepo.IsBlockedEither(ctx, fromUserID, toUserID)
	if err != nil {
		return uuid.Nil, err
	}
	if isBlocked {
		return uuid.Nil, errors.New("cannot message blocked user")
	}
	return toUserID, nil
}

func (h *ChatHandler) authenticate(c *gin.Context) (uuid.UUID, error) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("Authorization"))
	}
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	}
	if token == "" {
		return uuid.Nil, errors.New("missing token")
	}

	claims := &wsClaims{}
	t, err := jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !t.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	return claims.UserID, nil
}

func (h *ChatHandler) pingLoop(userID uuid.UUID, client *clientConn, stop <-chan struct{}) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = h.presenceRepo.UpsertLastSeen(context.Background(), userID, time.Now().UTC())
			if err := client.writeControl(gws.PingMessage, []byte("ping")); err != nil {
				return
			}
		case <-stop:
			return
		}
	}
}

func (h *ChatHandler) allowMessage(userID uuid.UUID) bool {
	const (
		maxMessagesPerWindow = 20
		windowSize           = 10 * time.Second
	)

	now := time.Now()

	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	s := h.rateByID[userID]
	if s.windowStart.IsZero() || now.Sub(s.windowStart) >= windowSize {
		h.rateByID[userID] = rateState{
			windowStart: now,
			count:       1,
		}
		return true
	}

	if s.count >= maxMessagesPerWindow {
		return false
	}

	s.count++
	h.rateByID[userID] = s
	return true
}
