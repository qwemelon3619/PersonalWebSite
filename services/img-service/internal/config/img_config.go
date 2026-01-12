package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"seungpyo.lee/PersonalWebSite/pkg/config"
)

type BlobConfig struct {
	config.GlobalConfig
	AzureStorageConnectionString string
	BlobContainerName            string
}

func LoadBlobConfig() *BlobConfig {
	// Load .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
	return &BlobConfig{
		GlobalConfig:                 *config.LoadGlobalConfig(),
		AzureStorageConnectionString: getEnv("AZURE_STORAGE_CONNECTION_STRING"),
		BlobContainerName:            getEnv("BLOB_CONTAINER_NAME"),
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
