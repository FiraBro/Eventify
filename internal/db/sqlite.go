package db

import (
	"database/sql"
	"github.com/FiraBro/local-go/internal/config"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", config.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatal("SQLite ping failed:", err)
	}

	if err := createEventTable(); err != nil {
		log.Fatal("Failed to create events table:", err)
	}

	log.Println("✅ SQLite connected")
}

func createEventTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		location TEXT NOT NULL,
		user_id TEXT,
		date_time DATETIME
	);`
	_, err := DB.Exec(query)
	return err
}
