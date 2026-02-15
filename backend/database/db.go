package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dbPath string) {
	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	createTables()
	log.Println("Database initialized at", dbPath)
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS saves (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			rom_name TEXT NOT NULL,
			save_name TEXT NOT NULL,
			save_data BLOB,
			is_nuzlocke BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS nuzlocke_pokemon (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			save_id INTEGER NOT NULL REFERENCES saves(id) ON DELETE CASCADE,
			pokemon_name TEXT NOT NULL,
			nickname TEXT,
			route TEXT NOT NULL,
			is_alive BOOLEAN DEFAULT TRUE,
			cause_of_death TEXT,
			level_caught INTEGER,
			notes TEXT
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatal("Failed to create table:", err)
		}
	}
}
