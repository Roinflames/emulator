package migrations

import "embed"

// FS contains the SQL migration files embedded into the binary.
//
// Naming convention: 0001_description.sql (numeric prefix controls order).
// Only files matching that pattern will be applied.
//
//go:embed *.sql
var FS embed.FS

