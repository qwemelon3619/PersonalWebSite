package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"seungpyo.lee/PersonalWebSite/pkg/config"
	"seungpyo.lee/PersonalWebSite/pkg/jwt"
)

// AuthConfig extends GlobalConfig with any auth-service specific configurations.
type GatewayConfig struct {
	config.GlobalConfig
	AuthServiceURL string
	PostServiceURL string
	ImgServiceURL  string
	JWTSecretKey   string
}

func LoadGatewayConfig() *GatewayConfig {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
	secret := getEnv("JWT_SECRET_KEY")
	if err := jwt.ValidateSecretStrength(secret); err != nil {
		log.Fatalf("invalid JWT_SECRET_KEY: %v", err)
	}
	return &GatewayConfig{
		GlobalConfig:   *config.LoadGlobalConfig(),
		AuthServiceURL: getEnv("AUTH_SERVICE_URL"),
		PostServiceURL: getEnv("POST_SERVICE_URL"),
		ImgServiceURL:  getEnv("IMG_SERVICE_URL"),
		JWTSecretKey:   secret,
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
