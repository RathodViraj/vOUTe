package config

type TimescaleDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func LoadTimescaleDBConfig() TimescaleDBConfig {
	return TimescaleDBConfig{
		Host:     GetEnvRequired("POSTGRES_HOST"),
		Port:     GetEnvAsInt("POSTGRES_PORT", 5432),
		User:     GetEnvRequired("POSTGRES_USER"),
		Password: GetEnvRequired("POSTGRES_PASSWORD"),
		DBName:   GetEnvWithDefault("POSTGRES_DB", "voute_timescale"),
	}

}
