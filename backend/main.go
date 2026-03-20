package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"pokemon-web/database"
	"pokemon-web/handlers"
	"pokemon-web/migrations"
	"pokemon-web/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	_ "github.com/lib/pq"
)

func main() {
	var (
		flagMigrate  = flag.Bool("migrate", false, "run pending migrations and exit")
		flagRollback = flag.Int("rollback", 0, "rollback last N migrations and exit")
		flagStatus   = flag.Bool("status", false, "print migration status and exit")
	)
	flag.Parse()

	// Resolve paths relative to executable
	execDir, _ := os.Getwd()
	romsDir := filepath.Join(execDir, "..", "roms")
	frontendDir := filepath.Join(execDir, "..", "frontend")

	// Allow overrides via env
	if v := os.Getenv("ROMS_DIR"); v != "" {
		romsDir = v
	}
	if v := os.Getenv("FRONTEND_DIR"); v != "" {
		frontendDir = v
	}

	// Database URL (PostgreSQL)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Migration-only commands.
	if *flagMigrate || *flagRollback > 0 || *flagStatus {
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatal("Failed to open database:", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			log.Fatal("Failed to ping database:", err)
		}

		var migFS fs.FS = migrations.FS
		if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
			migFS = os.DirFS(dir)
		}

		switch {
		case *flagStatus:
			st, err := database.Status(db, migFS)
			if err != nil {
				log.Fatal("Failed to get migration status:", err)
			}
			fmt.Printf("Applied: %v\n", st.Applied)
			fmt.Printf("Pending: %v\n", st.Pending)
			return
		case *flagRollback > 0:
			if err := database.Rollback(db, migFS, *flagRollback); err != nil {
				log.Fatal("Failed to rollback migrations:", err)
			}
			return
		case *flagMigrate:
			if err := database.Migrate(db, migFS); err != nil {
				log.Fatal("Failed to run migrations:", err)
			}
			return
		default:
			return
		}
	}

	// Ensure roms directory exists
	os.MkdirAll(romsDir, 0755)

	// Init database
	database.Init(dbURL)

	// Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: 64 * 1024 * 1024, // 64MB for NDS roms and save files
	})

	app.Use(cors.New())

	// API routes
	api := app.Group("/api")

	// Auth (public)
	api.Post("/register", handlers.Register)
	api.Post("/login", handlers.Login)

	// ROM download is public (EmulatorJS needs direct access)
	api.Get("/roms/download", handlers.ServeRom)

	// Protected routes
	protected := api.Group("", middleware.AuthRequired())

	// ROMs
	protected.Get("/roms", handlers.ListRoms)
	protected.Post("/roms/upload", handlers.UploadRom)

	// Saves
	protected.Get("/saves", handlers.ListSaves)
	protected.Post("/saves", handlers.CreateSave)
	protected.Get("/saves/:id", handlers.GetSaveData)
	protected.Put("/saves/:id", handlers.UpdateSaveData)
	protected.Delete("/saves/:id", handlers.DeleteSave)

	// Nuzlocke
	protected.Get("/saves/:id/nuzlocke", handlers.GetNuzlockeData)
	protected.Post("/saves/:id/nuzlocke", handlers.AddNuzlockePokemon)
	protected.Put("/saves/:id/nuzlocke/:pokemonId", handlers.UpdateNuzlockePokemon)
	protected.Put("/saves/:id/nuzlocke/:pokemonId/kill", handlers.KillNuzlockePokemon)
	protected.Delete("/saves/:id/nuzlocke/:pokemonId", handlers.DeleteNuzlockePokemon)
	protected.Get("/saves/:id/nuzlocke/routes", handlers.GetNuzlockeRoutes)
	protected.Post("/saves/:id/nuzlocke/routes", handlers.UpsertNuzlockeRoute)
	protected.Put("/saves/:id/nuzlocke/routes/:routeId", handlers.UpdateNuzlockeRoute)
	protected.Delete("/saves/:id/nuzlocke/routes/:routeId", handlers.DeleteNuzlockeRoute)

	// Serve frontend static files
	app.Static("/", frontendDir)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3030"
	}

	log.Printf("Pokemon Web running on http://localhost:%s", port)
	log.Printf("ROMs directory: %s", romsDir)
	log.Fatal(app.Listen(":" + port))
}
