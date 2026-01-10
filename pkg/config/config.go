package config

import (
	"os"
)

type GlobalConfig struct {
	AccessTokenTTL  int // in minutes
	RefreshTokenTTL int // in minutes
	ServerPort      string
}

func LoadGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		AccessTokenTTL:  30,   // in minutes
		RefreshTokenTTL: 1440, // in minutes (1 day)
		ServerPort:      getEnv("SERVER_PORT"),
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
