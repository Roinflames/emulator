package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(databaseURL string) {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	createTables()
	log.Println("Database connected successfully")
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS saves (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			rom_name TEXT NOT NULL,
			save_name TEXT NOT NULL,
			save_data BYTEA,
			is_nuzlocke BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS nuzlocke_pokemon (
			id SERIAL PRIMARY KEY,
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
