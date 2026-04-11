package config

type RedisConfig struct {
	URL string
	DB  int
}

func LoadRedisConfig() RedisConfig {
	redisURL := GetEnvWithDefault("REDIS_URL", "redis://localhost:6379")
	db := GetEnvAsInt("REDIS_DB", 0)

	return RedisConfig{
		URL: redisURL,
		DB:  db,
	}
}
