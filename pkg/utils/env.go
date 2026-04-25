package utils

import "os"

func GetEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func GetEnvOrPanic(key string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	panic("Environment variable " + key + " is required but not set")
}
