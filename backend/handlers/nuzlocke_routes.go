package handlers

import (
	"strconv"
	"strings"

	"pokemon-web/database"
	"pokemon-web/models"

	"github.com/gofiber/fiber/v2"
)

var allowedNuzlockeRouteStatuses = map[string]bool{
	"pending":  true,
	"visited":  true,
	"captured": true,
	"missed":   true,
}

func normalizeNuzlockeRouteStatus(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "pending"
	}
	if allowedNuzlockeRouteStatuses[v] {
		return v
	}
	return ""
}

func GetNuzlockeRoutes(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	rows, err := database.DB.Query(
		`SELECT id, save_id, route, status, notes
		 FROM nuzlocke_routes
		 WHERE save_id = $1
		 ORDER BY route`, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to query routes"})
	}
	defer rows.Close()

	var routes []models.NuzlockeRoute
	for rows.Next() {
		var r models.NuzlockeRoute
		var notes *string
		if err := rows.Scan(&r.ID, &r.SaveID, &r.Route, &r.Status, &notes); err != nil {
			continue
		}
		if notes != nil {
			r.Notes = *notes
		}
		routes = append(routes, r)
	}

	if routes == nil {
		routes = []models.NuzlockeRoute{}
	}
	return c.JSON(routes)
}

func UpsertNuzlockeRoute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var body struct {
		Route  string `json:"route"`
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	body.Route = strings.TrimSpace(body.Route)
	if body.Route == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "route is required"})
	}
	status := normalizeNuzlockeRouteStatus(body.Status)
	if status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	var r models.NuzlockeRoute
	r.SaveID = saveID
	var notes *string
	err = database.DB.QueryRow(
		`INSERT INTO nuzlocke_routes (save_id, route, status, notes, updated_at)
		 VALUES ($1, $2, $3, NULLIF($4, ''), NOW())
		 ON CONFLICT (save_id, route)
		 DO UPDATE SET status = EXCLUDED.status,
		              notes = COALESCE(EXCLUDED.notes, nuzlocke_routes.notes),
		              updated_at = NOW()
		 RETURNING id, save_id, route, status, notes`,
		saveID, body.Route, status, body.Notes,
	).Scan(&r.ID, &r.SaveID, &r.Route, &r.Status, &notes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to upsert route"})
	}
	if notes != nil {
		r.Notes = *notes
	}

	return c.JSON(r)
}

func UpdateNuzlockeRoute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}
	routeID, err := strconv.Atoi(c.Params("routeId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid route id"})
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	var body struct {
		Route  string `json:"route"`
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	body.Route = strings.TrimSpace(body.Route)
	if body.Route == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "route is required"})
	}
	status := normalizeNuzlockeRouteStatus(body.Status)
	if status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	var r models.NuzlockeRoute
	var notes *string
	err = database.DB.QueryRow(
		`UPDATE nuzlocke_routes
		 SET route = $1,
		     status = $2,
		     notes = NULLIF($3, ''),
		     updated_at = NOW()
		 WHERE id = $4 AND save_id = $5
		 RETURNING id, save_id, route, status, notes`,
		body.Route, status, body.Notes, routeID, saveID,
	).Scan(&r.ID, &r.SaveID, &r.Route, &r.Status, &notes)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "route not found"})
	}
	if notes != nil {
		r.Notes = *notes
	}

	return c.JSON(r)
}

func DeleteNuzlockeRoute(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}
	routeID, err := strconv.Atoi(c.Params("routeId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid route id"})
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM saves WHERE id = $1 AND user_id = $2)", saveID, userID).Scan(&exists)
	if err != nil || !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	result, err := database.DB.Exec("DELETE FROM nuzlocke_routes WHERE id = $1 AND save_id = $2", routeID, saveID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete route"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "route not found"})
	}

	return c.JSON(fiber.Map{"message": "route deleted"})
}

