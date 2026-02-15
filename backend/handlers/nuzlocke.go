package handlers

import (
	"strconv"

	"pokemon-web/database"
	"pokemon-web/models"

	"github.com/gofiber/fiber/v2"
)

func GetNuzlockeData(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	// Verify save belongs to user and is nuzlocke
	var isNuzlocke bool
	err = database.DB.QueryRow("SELECT is_nuzlocke FROM saves WHERE id = ? AND user_id = ?", saveID, userID).Scan(&isNuzlocke)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	rows, err := database.DB.Query(
		`SELECT id, save_id, pokemon_name, nickname, route, is_alive, cause_of_death, level_caught, notes
		 FROM nuzlocke_pokemon WHERE save_id = ? ORDER BY id`, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to query nuzlocke data"})
	}
	defer rows.Close()

	var pokemon []models.NuzlockePokemon
	for rows.Next() {
		var p models.NuzlockePokemon
		var nickname, causeOfDeath, notes *string
		var levelCaught *int
		if err := rows.Scan(&p.ID, &p.SaveID, &p.PokemonName, &nickname, &p.Route, &p.IsAlive, &causeOfDeath, &levelCaught, &notes); err != nil {
			continue
		}
		if nickname != nil {
			p.Nickname = *nickname
		}
		if causeOfDeath != nil {
			p.CauseOfDeath = *causeOfDeath
		}
		if levelCaught != nil {
			p.LevelCaught = *levelCaught
		}
		if notes != nil {
			p.Notes = *notes
		}
		pokemon = append(pokemon, p)
	}

	if pokemon == nil {
		pokemon = []models.NuzlockePokemon{}
	}
	return c.JSON(pokemon)
}

func AddNuzlockePokemon(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	// Verify ownership
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = ? AND user_id = ?)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var p models.NuzlockePokemon
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if p.PokemonName == "" || p.Route == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "pokemon_name and route are required"})
	}

	result, err := database.DB.Exec(
		`INSERT INTO nuzlocke_pokemon (save_id, pokemon_name, nickname, route, is_alive, level_caught, notes)
		 VALUES (?, ?, ?, ?, TRUE, ?, ?)`,
		saveID, p.PokemonName, p.Nickname, p.Route, p.LevelCaught, p.Notes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add pokemon"})
	}

	id, _ := result.LastInsertId()
	p.ID = int(id)
	p.SaveID = saveID
	p.IsAlive = true
	return c.Status(fiber.StatusCreated).JSON(p)
}

func KillNuzlockePokemon(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}
	pokemonID, err := strconv.Atoi(c.Params("pokemonId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pokemon id"})
	}

	// Verify ownership
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = ? AND user_id = ?)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var body struct {
		CauseOfDeath string `json:"cause_of_death"`
	}
	c.BodyParser(&body)

	result, err := database.DB.Exec(
		"UPDATE nuzlocke_pokemon SET is_alive = FALSE, cause_of_death = ? WHERE id = ? AND save_id = ?",
		body.CauseOfDeath, pokemonID, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update pokemon"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
	}

	return c.JSON(fiber.Map{"message": "pokemon marked as dead"})
}

func DeleteNuzlockePokemon(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}
	pokemonID, err := strconv.Atoi(c.Params("pokemonId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pokemon id"})
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = ? AND user_id = ?)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	result, err := database.DB.Exec("DELETE FROM nuzlocke_pokemon WHERE id = ? AND save_id = ?", pokemonID, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete pokemon"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
	}

	return c.JSON(fiber.Map{"message": "pokemon deleted"})
}
