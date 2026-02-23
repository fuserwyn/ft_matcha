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
)

type wsClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type ChatHandler struct {
	hub         *Hub
	likeRepo    *repository.LikeRepository
	messageRepo *repository.MessageRepository
	jwtSecret   string
	upgrader    gws.Upgrader

	rateMu   sync.Mutex
	rateByID map[uuid.UUID]rateState
}

type rateState struct {
	windowStart time.Time
	count       int
}

type incomingMessage struct {
	ToUserID string `json:"to_user_id"`
	Content  string `json:"content"`
}

func NewChatHandler(
	hub *Hub,
	likeRepo *repository.LikeRepository,
	messageRepo *repository.MessageRepository,
	jwtSecret string,
) *ChatHandler {
	return &ChatHandler{
		hub:         hub,
		likeRepo:    likeRepo,
		messageRepo: messageRepo,
		jwtSecret:   jwtSecret,
		rateByID:    make(map[uuid.UUID]rateState),
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
	go h.pingLoop(client, stopPing)

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
	if !h.allowMessage(fromUserID) {
		return errors.New("rate limit exceeded: too many messages")
	}

	toUserID, err := uuid.Parse(strings.TrimSpace(in.ToUserID))
	if err != nil {
		return errors.New("invalid to_user_id")
	}
	if toUserID == fromUserID {
		return errors.New("cannot message yourself")
	}

	content := strings.TrimSpace(in.Content)
	if len(content) == 0 || len(content) > 2000 {
		return errors.New("content must be 1-2000 characters")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	isMatch, err := h.likeRepo.IsMatch(ctx, fromUserID, toUserID)
	if err != nil {
		return err
	}
	if !isMatch {
		return errors.New("can only message matches")
	}

	msg, err := h.messageRepo.Create(ctx, fromUserID, toUserID, content)
	if err != nil {
		return err
	}

	event := gin.H{
		"type": "message",
		"data": gin.H{
			"id":          msg.ID,
			"sender_id":   msg.SenderID,
			"receiver_id": msg.ReceiverID,
			"content":     msg.Content,
			"created_at":  msg.CreatedAt,
		},
	}

	h.hub.SendToUser(fromUserID, event)
	h.hub.SendToUser(toUserID, event)
	return nil
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

func (h *ChatHandler) pingLoop(client *clientConn, stop <-chan struct{}) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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
