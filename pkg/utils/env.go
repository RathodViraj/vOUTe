package utils

import (
	"os"
	"strconv"
)

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

func GetEnvOrPanicInt(key string) int {
	val := GetEnvOrPanic(key)
	intVal, err := strconv.Atoi(val)
	if err != nil {
		panic("Environment variable " + key + " must be an integer, got: " + val)
	}
	return intVal
}
