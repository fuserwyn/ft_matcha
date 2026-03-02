package handlers

import (
	"fmt"
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
	authSvc         *services.AuthService
	syncSvc         *services.SyncService
	mailer          *services.Mailer
	secret          string
	publicAPIBase   string
	frontendBaseURL string
}

func NewAuthHandler(
	authSvc *services.AuthService,
	syncSvc *services.SyncService,
	mailer *services.Mailer,
	jwtSecret string,
	publicAPIBase string,
	frontendBaseURL string,
) *AuthHandler {
	return &AuthHandler{
		authSvc:         authSvc,
		syncSvc:         syncSvc,
		mailer:          mailer,
		secret:          jwtSecret,
		publicAPIBase:   publicAPIBase,
		frontendBaseURL: frontendBaseURL,
	}
}

type RegisterReq struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordReq struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordReq struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type UpdateAccountReq struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
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
	verifyToken, err := h.issueEmailVerificationToken(u.ID)
	if err != nil {
		log.Printf("[auth] failed issuing verify token for user=%s: %v", u.ID, err)
	} else {
		verifyLink := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", h.publicAPIBase, verifyToken)
		body := fmt.Sprintf("Hi %s,\n\nVerify your email by opening this link:\n%s\n", u.FirstName, verifyLink)
		if err := h.mailer.Send(u.Email, "Matcha email verification", body); err != nil {
			log.Printf("[auth] failed sending verify email to user=%s: %v", u.ID, err)
		}
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

// VerifyEmail godoc
// @Summary	Verify email
// @Tags		auth
// @Produce	json
// @Param		token	query		string	true	"Email verification token"
// @Success	200		{object}	map[string]string
// @Failure	400		{object}	map[string]string
// @Failure	401		{object}	map[string]string
// @Router		/api/v1/auth/verify-email [get]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	userID, err := h.parseEmailVerificationToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification token"})
		return
	}
	if err := h.authSvc.VerifyEmail(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email verified"})
}

// ForgotPassword godoc
// @Summary	Forgot password
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		body	body		ForgotPasswordReq	true	"Forgot password payload"
// @Success	200		{object}	map[string]string
// @Failure	400		{object}	map[string]string
// @Router		/api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resetToken, u, err := h.authSvc.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if u != nil && resetToken != "" {
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", h.frontendBaseURL, resetToken)
		body := fmt.Sprintf("Hi %s,\n\nReset your password by opening this link:\n%s\n", u.FirstName, resetLink)
		if err := h.mailer.Send(u.Email, "Matcha password reset", body); err != nil {
			log.Printf("[auth] failed sending password reset email to user=%s: %v", u.ID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "if this email exists, reset link was sent"})
}

// ResetPassword godoc
// @Summary	Reset password
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		body	body		ResetPasswordReq	true	"Reset password payload"
// @Success	200		{object}	map[string]string
// @Failure	400		{object}	map[string]string
// @Router		/api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authSvc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		if err == services.ErrInvalidResetToken {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
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
		if err == services.ErrEmailNotVerified {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email not verified"})
			return
		}
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

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get(middleware.UserIDKey)
	id := userID.(uuid.UUID)
	var req UpdateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authSvc.UpdateAccount(c.Request.Context(), id, req.Username, req.Email, req.FirstName, req.LastName); err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "user exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.syncSvc.SyncUser(c.Request.Context(), id); err != nil {
		log.Printf("[auth] sync to ES failed for user=%s: %v", id, err)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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

type emailVerificationClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func (h *AuthHandler) issueEmailVerificationToken(userID uuid.UUID) (string, error) {
	claims := &emailVerificationClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Subject:   "email_verification",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(h.secret))
}

func (h *AuthHandler) parseEmailVerificationToken(token string) (uuid.UUID, error) {
	claims := &emailVerificationClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.secret), nil
	})
	if err != nil || !parsed.Valid || claims.Subject != "email_verification" {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	return claims.UserID, nil
}
