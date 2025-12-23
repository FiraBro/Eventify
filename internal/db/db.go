package db

import (
	"database/sql"
	"log"

	"github.com/FiraBro/local-go/internal/config"
	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	log.Println("📦 Connecting to PostgreSQL:", config.DBAddr)

	db, err := sql.Open("postgres", config.DBAddr)
	if err != nil {
		log.Fatal("❌ Failed to connect to PostgreSQL:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("❌ PostgreSQL ping failed:", err)
	}

	log.Println("✅ PostgreSQL connected successfully")
	return db
}
