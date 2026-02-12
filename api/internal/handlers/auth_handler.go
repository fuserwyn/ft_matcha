package handlers

import (
	"github.com/gin-gonic/gin"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"
)

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type AuthHandler struct {
	svc *services.AuthService
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	u, err := h.svc.Register(c.Request.Context(), in.Username, in.Email, in.Password, in.FirstName, in.LastName)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(409, gin.H{"error": "username or email exists"})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(201, gin.H{"user": userToMap(u)})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	u, err := h.svc.Login(c.Request.Context(), in.Username, in.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(200, gin.H{"user": userToMap(u)})
}

func userToMap(u *repository.User) map[string]any {
	return map[string]any{
		"id":         u.ID.String(),
		"username":   u.Username,
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
	}
}
