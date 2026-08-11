package database

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Two shipped bugs came from SQL drifting away from the schema: the supervisor
// selected from an `app_state` table that never existed, so the Minecraft
// service could not find its active project, and the Activity page selected
// `audit_log.created_at` when the column is `occurred_at`. Both compiled
// cleanly and only failed when a user reached them.
//
// This test prepares every SQL statement in the tree against a freshly
// migrated database. SQLite validates table and column names at prepare time,
// so a rename that misses a call site fails here instead of in production.

// sqlLiteral matches a backtick-quoted string that begins with a SQL verb.
var sqlLiteral = regexp.MustCompile("(?s)`\\s*(SELECT|INSERT|UPDATE|DELETE)\\b.*?`")

func TestEverySQLStatementMatchesTheSchema(t *testing.T) {
	root := sourceRoot(t)

	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "schema.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var checked, skipped int
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Vendored dependencies carry their own SQL and schemas.
			switch d.Name() {
			case "bin", "third_party", "testdata", "node_modules", ".git", "web":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)

		for _, m := range sqlLiteral.FindAllString(string(src), -1) {
			stmt := strings.TrimSpace(strings.Trim(m, "`"))

			// Statements assembled at runtime cannot be prepared as written.
			// A fragment like `SELECT ` + cols + ` FROM x` appears here as the
			// bare verb, so anything missing its target clause is a fragment
			// rather than a complete statement.
			if !isCompleteStatement(stmt) ||
				strings.Contains(stmt, "%s") || strings.Contains(stmt, "%d") {
				skipped++
				continue
			}

			s, err := db.Prepare(stmt)
			if err != nil {
				t.Errorf("%s: statement does not match the migrated schema: %v\n%s",
					rel, err, indent(stmt))
				continue
			}
			s.Close()
			checked++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	// A regexp that silently matched nothing would make this test useless.
	if checked < 30 {
		t.Fatalf("only %d statements were checked; the scan is probably broken", checked)
	}
	t.Logf("verified %d SQL statements against the schema (%d dynamic, skipped)", checked, skipped)
}

// The supervisor's own lookup deserves an explicit check, because a failure
// there means Minecraft never starts and the panel still looks healthy.
func TestActiveInstanceLookupWorks(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "schema.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	// This is the exact query cmdSupervisor runs at startup.
	var instID int64
	err = db.QueryRow(`SELECT instance_id FROM active_instance WHERE id=1`).Scan(&instID)
	if err != nil && err.Error() == "sql: no rows in result set" {
		t.Fatal("active_instance has no row 1; the supervisor would never find a project")
	}
	if err != nil && strings.Contains(err.Error(), "no such") {
		t.Fatalf("supervisor lookup does not match the schema: %v", err)
	}

	// And the table the old code used must genuinely be absent, so this test
	// would have caught the original bug rather than passing either way.
	if _, err := db.Prepare(`SELECT active_instance_id FROM app_state WHERE id=1`); err == nil {
		t.Error("app_state exists; update this test to reflect the real schema")
	}
}

// isCompleteStatement reports whether a literal is a whole statement rather
// than a fragment that gets concatenated with a column list at runtime.
func isCompleteStatement(stmt string) bool {
	u := strings.ToUpper(stmt)
	switch {
	case strings.HasPrefix(u, "SELECT"), strings.HasPrefix(u, "DELETE"):
		return strings.Contains(u, "FROM")
	case strings.HasPrefix(u, "INSERT"):
		return strings.Contains(u, "INTO")
	case strings.HasPrefix(u, "UPDATE"):
		return strings.Contains(u, "SET")
	}
	return false
}

func sourceRoot(t *testing.T) string {
	t.Helper()
	// This file lives at <source>/internal/database/.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("cannot locate the source root: %v", err)
	}
	return root
}

func indent(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString("    " + strings.TrimSpace(line) + "\n")
	}
	return b.String()
}
