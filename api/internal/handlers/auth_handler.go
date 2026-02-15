package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"matcha/api/internal/middleware"
	"matcha/api/internal/services"
	"matcha/api/internal/validation"
)

type AuthHandler struct {
	authSvc *services.AuthService
	syncSvc *services.SyncService
	secret  string
}

func NewAuthHandler(authSvc *services.AuthService, syncSvc *services.SyncService, jwtSecret string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, syncSvc: syncSvc, secret: jwtSecret}
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

// Register godoc
// @Summary	Register new user
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		body	body		RegisterReq	true	"Registration data"
// @Success	201	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Failure	409	{object}	map[string]string
// @Router		/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateName(req.FirstName, "first_name"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateName(req.LastName, "last_name"); err != nil {
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
	token, err := h.issueToken(u.ID)
	if err != nil {
		log.Printf("[auth] register ok but token issue failed for user=%s: %v", u.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	log.Printf("[auth] register ok: user=%s username=%q", u.ID, u.Username)
	if err := h.syncSvc.SyncUser(c.Request.Context(), u.ID); err != nil {
		log.Printf("[auth] sync to ES failed for user=%s: %v", u.ID, err)
	}
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

// Login godoc
// @Summary	Login
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		body	body		LoginReq	true	"Credentials"
// @Success	200	{object}	map[string]interface{}
// @Failure	400	{object}	map[string]string
// @Failure	401	{object}	map[string]string
// @Router		/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("[auth] login failed for username=%q: %v", req.Username, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token, err := h.issueToken(u.ID)
	if err != nil {
		log.Printf("[auth] login ok but token issue failed for user=%s: %v", u.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	log.Printf("[auth] login ok: user=%s username=%q", u.ID, u.Username)
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

// Me godoc
// @Summary	Get current user
// @Tags		auth
// @Security	BearerAuth
// @Produce	json
// @Success	200	{object}	map[string]interface{}
// @Failure	401	{object}	map[string]string
// @Failure	404	{object}	map[string]string
// @Router		/api/v1/auth/me [get]
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
