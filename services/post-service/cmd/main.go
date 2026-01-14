package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/handler"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/repository"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/service"
)

func main() {

	conf := config.LoadPostConfig()
	dsn := conf.PostgreConnectionString
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	// auto migration
	if err := db.AutoMigrate(&domain.Post{}); err != nil {
		log.Fatalf("failed to migrate db: %v", err)
	}

	repo := repository.NewPostRepository(db)
	svc := service.NewPostService(repo)
	h := handler.NewPostHandler(svc)

	r := gin.Default()

	r.GET("/posts", h.GetPosts)
	r.GET("/posts/:id", h.GetPost)
	r.POST("/posts", h.CreatePost)
	r.PUT("/posts/:id", h.UpdatePost)
	r.DELETE("/posts/:id", h.DeletePost)

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
