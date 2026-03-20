package database

import (
	"database/sql"
	"io/fs"
	"log"
	"os"

	"pokemon-web/migrations"

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

	var migFS fs.FS = migrations.FS
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		migFS = os.DirFS(dir)
	}

	if err := Migrate(DB, migFS); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}
	log.Println("Database connected successfully")
}
