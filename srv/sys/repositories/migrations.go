package repositories

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationsDir = "migrations"

// ApplyPendingMigrations aplica las migraciones .sql pendientes sobre esta instancia.
// Es el mismo runner que corre migrate() al arrancar; expuesto para el comando
// CLI "tarhiata migrate up".
func (r *SQLiteRepository) ApplyPendingMigrations() error {
	return ApplyPendingMigrations(r.db)
}

// PendingMigrations lista los archivos de migración que aún no se aplicaron.
func (r *SQLiteRepository) PendingMigrations() ([]string, error) {
	return PendingMigrations(r.db)
}

// AppliedMigrations lista los archivos de migración ya aplicados, en orden.
func (r *SQLiteRepository) AppliedMigrations() ([]string, error) {
	return AppliedMigrations(r.db)
}

// ApplyPendingMigrations aplica, en orden, los archivos migrations/migration-NNN-*.sql
// que todavía no están registrados en schema_migrations. Idempotente: correrla dos
// veces seguidas no reintenta nada. La llama migrate() al arrancar, y también el
// comando CLI "tarhiata migrate up".
func ApplyPendingMigrations(db *sql.DB) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	names, err := listMigrationFiles()
	if err != nil {
		return err
	}

	for _, name := range names {
		applied, err := isMigrationApplied(db, name)
		if err != nil {
			return fmt.Errorf("verificar migración %s: %w", name, err)
		}
		if applied {
			continue
		}
		if err := applyMigrationFile(db, name); err != nil {
			return fmt.Errorf("aplicar migración %s: %w", name, err)
		}
	}
	return nil
}

// PendingMigrations devuelve los nombres de archivo de migraciones que aún no se aplicaron,
// en el orden en que se aplicarían. La usa "tarhiata migrate status".
func PendingMigrations(db *sql.DB) ([]string, error) {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return nil, err
	}

	names, err := listMigrationFiles()
	if err != nil {
		return nil, err
	}

	var pending []string
	for _, name := range names {
		applied, err := isMigrationApplied(db, name)
		if err != nil {
			return nil, fmt.Errorf("verificar migración %s: %w", name, err)
		}
		if !applied {
			pending = append(pending, name)
		}
	}
	return pending, nil
}

// AppliedMigrations devuelve los nombres de archivo ya aplicados, en el orden en que se aplicaron.
// La usa "tarhiata migrate status".
func AppliedMigrations(db *sql.DB) ([]string, error) {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT filename FROM schema_migrations ORDER BY applied_at ASC")
	if err != nil {
		return nil, fmt.Errorf("listar migraciones aplicadas: %w", err)
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied = append(applied, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return fmt.Errorf("crear tabla schema_migrations: %w", err)
	}
	return nil
}

func isMigrationApplied(db *sql.DB, filename string) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", filename).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func listMigrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationFiles, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("leer directorio de migraciones embebido: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool {
		return migrationNumber(names[i]) < migrationNumber(names[j])
	})
	return names, nil
}

// migrationNumber extrae el NNN de "migration-NNN-descripcion.sql".
func migrationNumber(filename string) int {
	parts := strings.SplitN(filename, "-", 3)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}

func applyMigrationFile(db *sql.DB, name string) error {
	content, err := migrationFiles.ReadFile(migrationsDir + "/" + name)
	if err != nil {
		return fmt.Errorf("leer archivo embebido: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitSQLStatements(string(content)) {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("ejecutar statement: %w", err)
		}
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", name); err != nil {
		return fmt.Errorf("registrar migración aplicada: %w", err)
	}

	return tx.Commit()
}

// splitSQLStatements separa un archivo .sql en statements individuales por ';', tras
// quitar los comentarios de línea ('-- ...') para que un ';' dentro de un comentario no
// corte el statement a la mitad. Suficiente para las migraciones DDL/DML simples de
// este repo (sin ';' dentro de strings ni procedimientos almacenados); el driver sqlite
// usado no soporta múltiples statements en un solo Exec.
func splitSQLStatements(content string) []string {
	var withoutComments strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		withoutComments.WriteString(line)
		withoutComments.WriteByte('\n')
	}

	raw := strings.Split(withoutComments.String(), ";")
	var stmts []string
	for _, s := range raw {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		stmts = append(stmts, trimmed)
	}
	return stmts
}
