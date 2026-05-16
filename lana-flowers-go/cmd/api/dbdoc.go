package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	dbpkg "lana-flowers-go/internal/db"
)

type columnInfo struct {
	Name     string
	Type     string
	Nullable string
	Default  sql.NullString
}

type indexInfo struct {
	Name       string
	Definition string
}

func generateDatabaseMD() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return
	}

	db, err := dbpkg.ConnectDB(dsn)
	if err != nil {
		fmt.Printf("DATABASE.md: skip (db connect error: %v)\n", err)
		return
	}
	defer db.Close()

	tables, err := getTables(db)
	if err != nil {
		fmt.Printf("DATABASE.md: skip (query error: %v)\n", err)
		return
	}

	var sb strings.Builder
	sb.WriteString("# Database Schema\n\n")
	sb.WriteString("Auto-generated after `migrate`. Do not edit manually.\n\n")

	for _, table := range tables {
		columns, err := getColumns(db, table)
		if err != nil {
			continue
		}
		indexes, err := getIndexes(db, table)
		if err != nil {
			indexes = nil
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", table))
		sb.WriteString("| Column | Type | Nullable | Default |\n")
		sb.WriteString("|--------|------|----------|---------|\n")

		for _, col := range columns {
			def := ""
			if col.Default.Valid {
				def = col.Default.String
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", col.Name, col.Type, col.Nullable, def))
		}

		if len(indexes) > 0 {
			sb.WriteString("\n**Indexes:**\n")
			for _, idx := range indexes {
				sb.WriteString(fmt.Sprintf("- `%s`: %s\n", idx.Name, idx.Definition))
			}
		}

		sb.WriteString("\n")
	}

	if err := os.WriteFile("DATABASE.md", []byte(sb.String()), 0644); err != nil {
		fmt.Printf("DATABASE.md: write error: %v\n", err)
		return
	}

	fmt.Println("DATABASE.md: updated")
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
		  AND table_name != 'schema_migrations'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, nil
}

func getColumns(db *sql.DB, table string) ([]columnInfo, error) {
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []columnInfo
	for rows.Next() {
		var c columnInfo
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Default); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func getIndexes(db *sql.DB, table string) ([]indexInfo, error) {
	rows, err := db.Query(`
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1
		  AND indexname NOT LIKE '%_pkey'
		ORDER BY indexname
	`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []indexInfo
	for rows.Next() {
		var idx indexInfo
		if err := rows.Scan(&idx.Name, &idx.Definition); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}
	return indexes, nil
}
