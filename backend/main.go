package main

import (
	"log"
	"os"
	"path/filepath"

	"pokemon-web/database"
	"pokemon-web/handlers"
	"pokemon-web/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	// Resolve paths relative to executable
	execDir, _ := os.Getwd()
	romsDir := filepath.Join(execDir, "..", "roms")
	frontendDir := filepath.Join(execDir, "..", "frontend")
	dbPath := filepath.Join(execDir, "pokemon.db")

	// Allow overrides via env
	if v := os.Getenv("ROMS_DIR"); v != "" {
		romsDir = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		dbPath = v
	}

	// Ensure roms directory exists
	os.MkdirAll(romsDir, 0755)
	handlers.RomsDir = romsDir

	// Init database
	database.Init(dbPath)

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

	// ROMs listing
	protected.Get("/roms", handlers.ListRoms)

	// Saves
	protected.Get("/saves", handlers.ListSaves)
	protected.Post("/saves", handlers.CreateSave)
	protected.Get("/saves/:id", handlers.GetSaveData)
	protected.Put("/saves/:id", handlers.UpdateSaveData)
	protected.Delete("/saves/:id", handlers.DeleteSave)

	// Nuzlocke
	protected.Get("/saves/:id/nuzlocke", handlers.GetNuzlockeData)
	protected.Post("/saves/:id/nuzlocke", handlers.AddNuzlockePokemon)
	protected.Put("/saves/:id/nuzlocke/:pokemonId/kill", handlers.KillNuzlockePokemon)
	protected.Delete("/saves/:id/nuzlocke/:pokemonId", handlers.DeleteNuzlockePokemon)

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
