package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"seungpyo.lee/PersonalWebSite/pkg/config"
)

type PostConfig struct {
	config.GlobalConfig
	PostgreDBURL      string
	PostgreDBPort     string
	PostgreDBUser     string
	PostgreDBPassword string
	PostgreDBName     string
	RedisDBURL        string
	RedisDBPort       string
	RedisDBPassword   string
	RedisMaxRetries   int
	RedisPoolSize     int
}

func LoadPostConfig() *PostConfig {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
	return &PostConfig{
		GlobalConfig:      *config.LoadGlobalConfig(),
		PostgreDBURL:      getEnv("POSTGRE_DB_URL"),
		PostgreDBPort:     getEnv("POSTGRE_DB_PORT"),
		PostgreDBUser:     getEnv("POSTGRE_DB_USER"),
		PostgreDBPassword: getEnv("POSTGRE_DB_PASSWORD"),
		PostgreDBName:     getEnv("POSTGRE_DB_NAME"),
		RedisDBURL:        getEnv("REDIS_DB_URL"),
		RedisDBPort:       getEnv("REDIS_DB_PORT"),
		RedisDBPassword:   getEnv("REDIS_DB_PASSWORD"),
		RedisMaxRetries:   3,
		RedisPoolSize:     10,
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
