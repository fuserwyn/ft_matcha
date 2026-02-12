package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"matcha/api/internal/config"
	"matcha/api/internal/database"
	"matcha/api/internal/handlers"
	"matcha/api/internal/middleware"
	"matcha/api/internal/repository"
	"matcha/api/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()
	pool, err := database.NewPool(ctx, config.DatabaseURL())
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	log.Println("DB connected")

	if err := database.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("migration: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	profileRepo := repository.NewProfileRepository(pool)
	authSvc := services.NewAuthService(userRepo)

	authH := handlers.NewAuthHandler(authSvc, config.JWTSecret())
	profileH := handlers.NewProfileHandler(profileRepo)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.GET("/auth/me", middleware.Auth(config.JWTSecret()), authH.Me)

		profile := api.Group("/profile")
		profile.Use(middleware.Auth(config.JWTSecret()))
		{
			profile.GET("/me", profileH.GetMe)
			profile.PUT("/me", profileH.UpdateMe)
		}
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API :%s", port)
	r.Run(":" + port)
}
