package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/handler"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/repository"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/service"
)

func main() {

	conf := config.LoadAuthConfig()

	dsn := conf.PostgreConnectionString
	if dsn == "" {
		log.Fatal("POSTGRES_DSN environment variable is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Auto-migrate User model
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	repo := repository.NewUserRepository(db)
	redisUrl := fmt.Sprintf("%s:%s", conf.RedisDBURL, conf.RedisDBPort)
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: conf.RedisDBPassword,
		DB:       0, // use default DB
	})

	tokenManager := jwt.NewTokenManager(conf.JWTSecretKey, redisClient)
	svc := service.NewAuthService(repo, tokenManager)
	h := handler.NewAuthHandler(svc, conf)

	r := gin.Default()

	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.GET("/users/:id", h.GetUser)

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
