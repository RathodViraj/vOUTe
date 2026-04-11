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
		Host:     GetEnvWithDefault("TIMESCALE_DB_HOST", "localhost"),
		Port:     GetEnvAsInt("TIMESCALE_DB_PORT", 5432),
		User:     GetEnvWithDefault("TIMESCALE_DB_USER", "your_username"),
		Password: GetEnvWithDefault("TIMESCALE_DB_PASSWORD", "your_password"),
		DBName:   GetEnvWithDefault("TIMESCALE_DB_NAME", "your_dbname"),
	}

}
