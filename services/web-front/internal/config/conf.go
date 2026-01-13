package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"seungpyo.lee/PersonalWebSite/pkg/config"
)

type PostConfig struct {
	config.GlobalConfig
	BlobBaseUrl   string
	ApiGatewayURL string
}

func LoadWebConfig() *PostConfig {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
	return &PostConfig{
		GlobalConfig:  *config.LoadGlobalConfig(),
		BlobBaseUrl:   getEnv("IMG_BLOB_BASE_URL"),
		ApiGatewayURL: getEnv("API_GATEWAY_URL"),
	}
}

// getEnv retrieves the value of the environment variable named by the key.
func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	} else {
		panic("critical config missing: " + key)
	}
}
