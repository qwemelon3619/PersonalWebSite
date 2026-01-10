package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
