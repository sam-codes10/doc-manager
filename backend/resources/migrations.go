package resources

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunPostgresMigrations executes all .sql migration files in the PostgreSQL migrations folder
func RunPostgresMigrations(ctx context.Context, db *pgxpool.Pool, customPath ...string) error {
	dirPath := resolveMigrationDir(append(customPath, "migrations/postgres", "../migrations/postgres", "backend/migrations/postgres")...)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read postgres migrations directory %q: %w", dirPath, err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		fullPath := filepath.Join(dirPath, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read sql file %s: %w", file, err)
		}

		if _, err := db.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("failed executing migration %s: %w", file, err)
		}
		log.Printf("[Postgres Migration] Successfully applied: %s", file)
	}

	return nil
}

// RunCassandraMigrations executes all .cql migration files in the Cassandra migrations folder
func RunCassandraMigrations(ctx context.Context, session *gocql.Session, customPath ...string) error {
	dirPath := resolveMigrationDir(append(customPath, "migrations/cassandra", "../migrations/cassandra", "backend/migrations/cassandra")...)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read cassandra migrations directory %q: %w", dirPath, err)
	}

	var cqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".cql") {
			cqlFiles = append(cqlFiles, f.Name())
		}
	}
	sort.Strings(cqlFiles)

	for _, file := range cqlFiles {
		fullPath := filepath.Join(dirPath, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read cql file %s: %w", file, err)
		}

		// Split by semicolons because gocql executes single CQL statements
		statements := splitCQLStatements(string(content))
		for _, stmt := range statements {
			if err := session.Query(stmt).WithContext(ctx).Exec(); err != nil {
				return fmt.Errorf("failed executing CQL in %s: %w (statement: %s)", file, err, stmt)
			}
		}
		log.Printf("[Cassandra Migration] Successfully applied: %s", file)
	}

	return nil
}

func splitCQLStatements(content string) []string {
	var statements []string
	rawStmts := strings.Split(content, ";")
	for _, raw := range rawStmts {
		// Strip comments and clean whitespace
		var lines []string
		for _, line := range strings.Split(raw, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			lines = append(lines, line)
		}
		stmt := strings.TrimSpace(strings.Join(lines, "\n"))
		if stmt != "" {
			// gocql sessions are keyspace-scoped; skip USE statements
			if strings.HasPrefix(strings.ToUpper(stmt), "USE ") {
				continue
			}
			statements = append(statements, stmt)
		}
	}
	return statements
}

func resolveMigrationDir(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return "migrations"
}
