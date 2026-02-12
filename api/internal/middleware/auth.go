package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const UserIDKey = "user_id"

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			log.Printf("[auth] %s %s: missing or invalid Authorization header", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		claims := &Claims{}
		t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil {
			log.Printf("[auth] %s %s: token parse error: %v", c.Request.Method, c.Request.URL.Path, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		if !t.Valid {
			log.Printf("[auth] %s %s: token not valid", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if ah := c.GetHeader("Authorization"); ah != "" && strings.HasPrefix(ah, "Bearer ") {
		return strings.TrimPrefix(ah, "Bearer ")
	}
	return ""
}
