package db

import "embed"

// Migrations are applied in order when the server starts.
//
//go:embed migrations/*.sql
var Migrations embed.FS
