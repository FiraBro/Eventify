package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDb() {
	var err error

	DB, err = sql.Open("sqlite3", "./api.db")
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("✅ SQLite connected")

	createEventTable()
}

func createEventTable() {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		location TEXT NOT NULL,
		user_id TEXT,
		date_time TEXT NOT NULL
	);
	`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal(err)
	}
}
