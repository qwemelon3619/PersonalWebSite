package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/config"
)

func main() {
	conf := config.LoadGlobalConfig()
	r := gin.Default()

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
