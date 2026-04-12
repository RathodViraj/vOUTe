package db

import (
	"database/sql"
	"fmt"
	"voute/pkg/config"
)

func ConnectTimescaleDB() (*sql.DB, error) {
	cfg := config.LoadTimescaleDBConfig()

	dns := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := sql.Open("postgres", dns)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to TimescaleDB: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping TimescaleDB: %v", err)
	}

	return db, nil
}

func CloseTimescaleDB(db *sql.DB) error {
	return db.Close()
}
