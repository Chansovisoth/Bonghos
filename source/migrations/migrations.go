// Package migrations embeds the versioned SQLite schema migrations.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
