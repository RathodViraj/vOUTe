package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment")
	}
}

func GetEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return value
}

func GetEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Error converting %s to int: %v. Using default value: %d", key, err, defaultValue)
		return defaultValue
	}
	return value
}
