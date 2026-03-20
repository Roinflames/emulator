package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	migrationUpDownRe = regexp.MustCompile(`^(\d+)_.*\.(up|down)\.sql$`)
	migrationLegacyRe = regexp.MustCompile(`^(\d+)_.*\.sql$`) // treated as .up.sql
)

type migration struct {
	Version int64
	Name    string
	UpPath  string
	DownPath string
}

func Migrate(db *sql.DB, migrationsFS fs.FS) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	// Track applied migrations inside the DB.
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := loadAppliedVersions(db)
	if err != nil {
		return err
	}

	migs, err := discoverMigrations(migrationsFS)
	if err != nil {
		return err
	}

	for _, m := range migs {
		if applied[m.Version] {
			continue
		}

		if m.UpPath == "" {
			return fmt.Errorf("missing up migration for version %d (%s)", m.Version, m.Name)
		}

		sqlBytes, err := fs.ReadFile(migrationsFS, m.UpPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.UpPath, err)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			return fmt.Errorf("migration %s is empty", m.UpPath)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", m.UpPath, err)
		}

		if _, err := tx.Exec(sqlText); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.UpPath, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.Version, m.Name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.UpPath, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.UpPath, err)
		}
	}

	return nil
}

func Rollback(db *sql.DB, migrationsFS fs.FS, steps int) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if steps <= 0 {
		return fmt.Errorf("steps must be > 0")
	}

	// Ensure schema_migrations exists (safe if already there).
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migs, err := discoverMigrations(migrationsFS)
	if err != nil {
		return err
	}
	byVersion := map[int64]migration{}
	for _, m := range migs {
		byVersion[m.Version] = m
	}

	appliedVersions, err := loadAppliedVersionsOrderedDesc(db)
	if err != nil {
		return err
	}
	if len(appliedVersions) == 0 {
		return nil
	}

	if steps > len(appliedVersions) {
		steps = len(appliedVersions)
	}

	for i := 0; i < steps; i++ {
		v := appliedVersions[i]
		m, ok := byVersion[v]
		if !ok || m.DownPath == "" {
			return fmt.Errorf("missing down migration for version %d", v)
		}

		sqlBytes, err := fs.ReadFile(migrationsFS, m.DownPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.DownPath, err)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			return fmt.Errorf("migration %s is empty", m.DownPath)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", m.DownPath, err)
		}
		if _, err := tx.Exec(sqlText); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.DownPath, err)
		}
		if _, err := tx.Exec(`DELETE FROM schema_migrations WHERE version = $1`, v); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("unrecord migration %d: %w", v, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit rollback %s: %w", m.DownPath, err)
		}
	}

	return nil
}

type MigrationStatus struct {
	Applied []int64
	Pending []int64
}

func Status(db *sql.DB, migrationsFS fs.FS) (*MigrationStatus, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	appliedSet, err := loadAppliedVersions(db)
	if err != nil {
		return nil, err
	}
	appliedOrdered, err := loadAppliedVersionsOrderedDesc(db)
	if err != nil {
		return nil, err
	}
	// Normalize ascending for display.
	sort.Slice(appliedOrdered, func(i, j int) bool { return appliedOrdered[i] < appliedOrdered[j] })

	migs, err := discoverMigrations(migrationsFS)
	if err != nil {
		return nil, err
	}

	var pending []int64
	for _, m := range migs {
		if !appliedSet[m.Version] && m.UpPath != "" {
			pending = append(pending, m.Version)
		}
	}

	return &MigrationStatus{
		Applied: appliedOrdered,
		Pending: pending,
	}, nil
}

func loadAppliedVersions(db *sql.DB) (map[int64]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

func loadAppliedVersionsOrderedDesc(db *sql.DB) ([]int64, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version DESC`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return versions, nil
}

func discoverMigrations(migrationsFS fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	byVersion := map[int64]migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if m := migrationUpDownRe.FindStringSubmatch(name); m != nil {
			v, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse migration version from %q: %w", name, err)
			}
			cur := byVersion[v]
			cur.Version = v
			cur.Name = name // keep last-seen as display; later we normalize
			if m[2] == "up" {
				if cur.UpPath != "" {
					return nil, fmt.Errorf("duplicate up migration for version %d: %q and %q", v, cur.UpPath, name)
				}
				cur.UpPath = name
			} else {
				if cur.DownPath != "" {
					return nil, fmt.Errorf("duplicate down migration for version %d: %q and %q", v, cur.DownPath, name)
				}
				cur.DownPath = name
			}
			byVersion[v] = cur
			continue
		}

		// Legacy: treat as up migration.
		if m := migrationLegacyRe.FindStringSubmatch(name); m != nil {
			v, err := strconv.ParseInt(m[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse migration version from %q: %w", name, err)
			}
			cur := byVersion[v]
			cur.Version = v
			cur.Name = name
			if cur.UpPath != "" {
				// If both legacy and .up exist, that's ambiguous.
				return nil, fmt.Errorf("duplicate up migration for version %d: %q and %q", v, cur.UpPath, name)
			}
			cur.UpPath = name
			byVersion[v] = cur
			continue
		}
	}

	var migs []migration
	for _, m := range byVersion {
		// Prefer a stable "name" for display/recording: use the up filename if present.
		if m.UpPath != "" {
			m.Name = m.UpPath
		} else if m.DownPath != "" {
			m.Name = m.DownPath
		}
		migs = append(migs, m)
	}

	sort.Slice(migs, func(i, j int) bool {
		if migs[i].Version == migs[j].Version {
			return migs[i].Name < migs[j].Name
		}
		return migs[i].Version < migs[j].Version
	})

	return migs, nil
}
