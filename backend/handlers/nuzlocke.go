package handlers

import (
	"database/sql"
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

	// Verify save belongs to user
	var isNuzlocke bool
	err = database.DB.QueryRow("SELECT is_nuzlocke FROM saves WHERE id = $1 AND user_id = $2", saveID, userID).Scan(&isNuzlocke)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	rows, err := database.DB.Query(
		`SELECT id, save_id, pokemon_name, nickname, route, is_alive, cause_of_death, level_caught, notes
		 FROM nuzlocke_pokemon WHERE save_id = $1 ORDER BY id`, saveID)
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

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to start transaction"})
	}
	defer tx.Rollback()

	// Verify ownership
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
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

	var id int
	err = tx.QueryRow(
		`INSERT INTO nuzlocke_pokemon (save_id, pokemon_name, nickname, route, is_alive, level_caught, notes)
		 VALUES ($1, $2, $3, $4, TRUE, $5, $6) RETURNING id`,
		saveID, p.PokemonName, p.Nickname, p.Route, p.LevelCaught, p.Notes).Scan(&id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add pokemon"})
	}

	if err := ensureRouteCaptured(tx, saveID, p.Route); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sync route"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to commit transaction"})
	}

	p.ID = id
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
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var body struct {
		CauseOfDeath string `json:"cause_of_death"`
	}
	c.BodyParser(&body)

	result, err := database.DB.Exec(
		"UPDATE nuzlocke_pokemon SET is_alive = FALSE, cause_of_death = $1 WHERE id = $2 AND save_id = $3",
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

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to start transaction"})
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var route string
	err = tx.QueryRow("SELECT route FROM nuzlocke_pokemon WHERE id = $1 AND save_id = $2", pokemonID, saveID).Scan(&route)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load pokemon"})
	}

	result, err := tx.Exec("DELETE FROM nuzlocke_pokemon WHERE id = $1 AND save_id = $2", pokemonID, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete pokemon"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
	}

	if err := downgradeCapturedRouteIfEmpty(tx, saveID, route); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sync route"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to commit transaction"})
	}

	return c.JSON(fiber.Map{"message": "pokemon deleted"})
}

func UpdateNuzlockePokemon(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}
	pokemonID, err := strconv.Atoi(c.Params("pokemonId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid pokemon id"})
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to start transaction"})
	}
	defer tx.Rollback()

	// Verify ownership
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var oldRoute string
	err = tx.QueryRow("SELECT route FROM nuzlocke_pokemon WHERE id = $1 AND save_id = $2", pokemonID, saveID).Scan(&oldRoute)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load pokemon"})
	}

	var body struct {
		PokemonName string `json:"pokemon_name"`
		Nickname    string `json:"nickname"`
		Route       string `json:"route"`
		LevelCaught int    `json:"level_caught"`
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.PokemonName == "" || body.Route == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "pokemon_name and route are required"})
	}

	var p models.NuzlockePokemon
	var nickname, causeOfDeath, notes *string
	var levelCaught *int
	err = tx.QueryRow(
		`UPDATE nuzlocke_pokemon
		 SET pokemon_name = $1,
		     nickname = NULLIF($2, ''),
		     route = $3,
		     level_caught = CASE WHEN $4 <= 0 THEN NULL ELSE $4 END,
		     notes = NULLIF($5, '')
		 WHERE id = $6 AND save_id = $7
		 RETURNING id, save_id, pokemon_name, nickname, route, is_alive, cause_of_death, level_caught, notes`,
		body.PokemonName, body.Nickname, body.Route, body.LevelCaught, body.Notes, pokemonID, saveID,
	).Scan(&p.ID, &p.SaveID, &p.PokemonName, &nickname, &p.Route, &p.IsAlive, &causeOfDeath, &levelCaught, &notes)
	if err != nil {
		// If no rows are updated, Scan returns an error; treat as not found to avoid leaking DB details.
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "pokemon not found"})
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

	if err := ensureRouteCaptured(tx, saveID, body.Route); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sync route"})
	}
	if oldRoute != body.Route {
		if err := downgradeCapturedRouteIfEmpty(tx, saveID, oldRoute); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sync route"})
		}
	}
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to commit transaction"})
	}

	return c.JSON(p)
}
