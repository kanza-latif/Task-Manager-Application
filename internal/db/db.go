package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func LoadConfig() Config {
	return Config{
		Host:     getEnv("DB_HOST", "127.0.0.1"),
		Port:     getEnv("DB_PORT", "3306"),
		User:     getEnv("DB_USER", "root"),
		Password: getEnv("DB_PASSWORD", ""),
		Name:     getEnv("DB_NAME", "taskmanager"),
	}
}

func Connect(cfg Config) (*sql.DB, error) {
	// Step 1 — connect without selecting any database
	dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port,
	)

	rootDB, err := sql.Open("mysql", dsnNoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	defer rootDB.Close()

	if err := rootDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to reach db at %s:%s — %w", cfg.Host, cfg.Port, err)
	}

	// Step 2 — check if the database exists
	var dbName string
	row := rootDB.QueryRow(
		`SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?`, cfg.Name,
	)
	err = row.Scan(&dbName)

	if err == sql.ErrNoRows {
		// Database does not exist — create it
		fmt.Printf("Database '%s' not found. Creating it...\n", cfg.Name)
		_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE `%s`", cfg.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to create database '%s': %w", cfg.Name, err)
		}
		fmt.Printf("Database '%s' created successfully.\n", cfg.Name)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check database existence: %w", err)
	} else {
		fmt.Printf("Database '%s' found. Connecting...\n", cfg.Name)
	}

	// Step 3 — reconnect with the database selected
	dsnWithDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sql.Open("mysql", dsnWithDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to db '%s': %w", cfg.Name, err)
	}

	return db, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
