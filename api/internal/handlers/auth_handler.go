package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/services"
)

type AuthHandler struct {
	authSvc *services.AuthService
	secret  string
}

func NewAuthHandler(authSvc *services.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, secret: jwtSecret}
}

type RegisterReq struct {
	Username   string `json:"username" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.authSvc.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "user exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, _ := h.issueToken(u.ID)
	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, _ := h.issueToken(u.ID)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	u, err := h.authSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
	})
}

func (h *AuthHandler) issueToken(userID uuid.UUID) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.secret))
}
