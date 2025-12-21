package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB is a global database connection
var DB *sql.DB

// InitDb initializes the SQLite database
func InitDb() {
	var err error

	// Open (or create) SQLite database file "api.db"
	DB, err = sql.Open("sqlite3", "./api.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test the connection
	err = DB.Ping()
	if err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("✅ SQLite database connected successfully")

 createEventTable()	
}


func createEventTable(){
createTable := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		location TEXT NOT NULL,
		user_id TEXT,
		date_time TEXT
	);
	`

	_, err := DB.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create events table:", err)
	}
}