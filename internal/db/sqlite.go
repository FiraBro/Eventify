package db

import (
	"database/sql"
	"log"

	"github.com/FiraBro/local-go/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes SQLite and returns the DB connection
func InitDB() *sql.DB {
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		log.Fatal("❌ Failed to connect to SQLite:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("❌ SQLite ping failed:", err)
	}

	// Run migrations
	if err := createUsersTable(db); err != nil {
		log.Fatal("❌ Failed to create users table:", err)
	}

	if err := createEventsTable(db); err != nil {
		log.Fatal("❌ Failed to create events table:", err)
	}

	log.Println("✅ SQLite connected and migrations applied")
	return db
}

// ----------------------------
// CREATE TABLES
// ----------------------------
func createUsersTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	return err
}

func createEventsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		location TEXT NOT NULL,
		user_id TEXT NOT NULL,
		date_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`
	_, err := db.Exec(query)
	return err
}
