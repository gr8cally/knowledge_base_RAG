package sqlite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Initialize ensures the SQLite file exists and applies any pending SQL migrations.
func Initialize(ctx context.Context, dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create sqlite dir: %w", err)
	}

	if _, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o644); err != nil {
		return fmt.Errorf("open sqlite file: %w", err)
	}

	if err := execSQL(ctx, dbPath, `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS schema_migrations (
  name TEXT PRIMARY KEY,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		return err
	}

	return runMigrations(ctx, dbPath, "migrations")
}

// Ping verifies the sqlite3 CLI can open the database and execute a trivial query.
func Ping(ctx context.Context, dbPath string) error {
	return execSQL(ctx, dbPath, "PRAGMA foreign_keys = ON; SELECT 1;")
}

func runMigrations(ctx context.Context, dbPath, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var filenames []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		filenames = append(filenames, entry.Name())
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		applied, err := migrationApplied(ctx, dbPath, filename)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}

		script := strings.Builder{}
		script.WriteString("BEGIN;\nPRAGMA foreign_keys = ON;\n")
		script.Write(sqlBytes)
		script.WriteString("\nINSERT INTO schema_migrations (name) VALUES (")
		script.WriteString(sqlQuote(filename))
		script.WriteString(");\nCOMMIT;\n")

		if err := execSQL(ctx, dbPath, script.String()); err != nil {
			return fmt.Errorf("apply migration %s: %w", filename, err)
		}
	}

	return nil
}

func migrationApplied(ctx context.Context, dbPath, filename string) (bool, error) {
	query := "SELECT COUNT(1) FROM schema_migrations WHERE name = " + sqlQuote(filename) + ";"
	out, err := execSQLWithOutput(ctx, dbPath, query)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "1", nil
}

func execSQL(ctx context.Context, dbPath, script string) error {
	_, err := execSQLWithOutput(ctx, dbPath, script)
	return err
}

func execSQLWithOutput(ctx context.Context, dbPath, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "sqlite3", dbPath)
	cmd.Stdin = strings.NewReader(script)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("sqlite3 error: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
