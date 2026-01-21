package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"seungpyo.lee/PersonalWebSite/pkg/config"
)

type PostConfig struct {
	config.GlobalConfig
	PostgreConnectionString string
	RedisDBURL              string
	RedisDBPort             string
	RedisDBPassword         string
	RedisMaxRetries         int
	RedisPoolSize           int
	ApiGatewayURL           string // URL of the API Gateway
	TranslationAPIURL       string // optional translation service URL
	TranslationAPIKey       string // optional translation service API key (e.g., DeepL)
}

func LoadPostConfig() *PostConfig {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
	return &PostConfig{
		GlobalConfig:            *config.LoadGlobalConfig(),
		PostgreConnectionString: getEnv("POSTGRE_CONNECTION_STRING"),
		RedisDBURL:              getEnv("REDIS_DB_URL"),
		RedisDBPort:             getEnv("REDIS_DB_PORT"),
		RedisDBPassword:         getEnv("REDIS_DB_PASSWORD"),
		ApiGatewayURL:           getEnv("API_GATEWAY_URL"),
		TranslationAPIURL:       getEnv("TRANSLATION_API_URL"),
		TranslationAPIKey:       getEnv("TRANSLATION_API_KEY"),
		RedisMaxRetries:         3,
		RedisPoolSize:           10,
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

// getEnvOrDefault retrieves the value or returns default if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
