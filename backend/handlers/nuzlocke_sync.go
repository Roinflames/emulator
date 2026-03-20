package handlers

import (
	"database/sql"
	"strings"
)

type nuzlockeDB interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func ensureRouteCaptured(db nuzlockeDB, saveID int, route string) error {
	route = strings.TrimSpace(route)
	if route == "" {
		return nil
	}

	_, err := db.Exec(
		`INSERT INTO nuzlocke_routes (save_id, route, status, updated_at)
		 VALUES ($1, $2, 'captured', NOW())
		 ON CONFLICT (save_id, route)
		 DO UPDATE SET status = 'captured', updated_at = NOW()`,
		saveID, route,
	)
	return err
}

func downgradeCapturedRouteIfEmpty(db nuzlockeDB, saveID int, route string) error {
	route = strings.TrimSpace(route)
	if route == "" {
		return nil
	}

	var cnt int
	if err := db.QueryRow(
		"SELECT COUNT(1) FROM nuzlocke_pokemon WHERE save_id = $1 AND route = $2",
		saveID, route,
	).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}

	_, err := db.Exec(
		`UPDATE nuzlocke_routes
		 SET status = 'visited', updated_at = NOW()
		 WHERE save_id = $1 AND route = $2 AND status = 'captured'`,
		saveID, route,
	)
	return err
}
