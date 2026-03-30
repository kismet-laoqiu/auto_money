package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Migration struct {
	Name string
	SQL  string
}

type HealthStatus struct {
	OK                    bool   `json:"ok"`
	Database              string `json:"database"`
	Version               string `json:"version"`
	PingMS                int64  `json:"ping_ms"`
	MaxOpenConns          int    `json:"max_open_conns"`
	MaxIdleConns          int    `json:"max_idle_conns"`
	RetentionDays         int    `json:"retention_days"`
	CompressionAfterHours int    `json:"compression_after_hours"`
}

func Open(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no sql migrations found in %s", dir)
	}
	migrations := make([]Migration, 0, len(names))
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		migrations = append(migrations, Migration{Name: name, SQL: strings.TrimSpace(string(body))})
	}
	return migrations, nil
}

func Migrate(ctx context.Context, cfg Config, dir string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db, err := Open(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}
	applied, err := loadAppliedNames(ctx, db)
	if err != nil {
		return nil, err
	}
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return nil, err
	}

	appliedNow := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		if applied[migration.Name] {
			continue
		}
		if _, err := db.ExecContext(ctx, migration.SQL); err != nil {
			return nil, fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}
		if _, err := db.ExecContext(ctx, `
            INSERT INTO warehouse_schema_migrations (name, applied_at)
            VALUES ($1, NOW())
        `, migration.Name); err != nil {
			return nil, fmt.Errorf("record migration %s: %w", migration.Name, err)
		}
		appliedNow = append(appliedNow, migration.Name)
	}
	if err := ApplyPolicies(ctx, db, cfg); err != nil {
		return nil, err
	}
	return appliedNow, nil
}

func Health(ctx context.Context, cfg Config) (HealthStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && cfg.HealthTimeout > 0 {
		next, cancel := context.WithTimeout(ctx, cfg.HealthTimeout)
		defer cancel()
		ctx = next
	}
	db, err := Open(cfg)
	if err != nil {
		return HealthStatus{}, err
	}
	defer db.Close()

	started := time.Now()
	if err := db.PingContext(ctx); err != nil {
		return HealthStatus{}, err
	}
	var version string
	if err := db.QueryRowContext(ctx, `SHOW server_version`).Scan(&version); err != nil {
		return HealthStatus{}, err
	}
	var database string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&database); err != nil {
		return HealthStatus{}, err
	}
	return HealthStatus{
		OK:                    true,
		Database:              database,
		Version:               version,
		PingMS:                time.Since(started).Milliseconds(),
		MaxOpenConns:          cfg.MaxOpenConns,
		MaxIdleConns:          cfg.MaxIdleConns,
		RetentionDays:         cfg.RetentionDays,
		CompressionAfterHours: cfg.CompressionAfterHours,
	}, nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS warehouse_schema_migrations (
            name TEXT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL
        )
    `)
	return err
}

func loadAppliedNames(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM warehouse_schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func ApplyPolicies(ctx context.Context, db *sql.DB, cfg Config) error {
	if cfg.CompressionAfterHours > 0 {
		if _, err := db.ExecContext(ctx, `
            ALTER TABLE market_bars
            SET (
                timescaledb.compress,
                timescaledb.compress_segmentby = 'provider,symbol,interval'
            )
        `); err != nil {
			return fmt.Errorf("enable compression: %w", err)
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf(
			"SELECT add_compression_policy('market_bars', INTERVAL '%d hours', if_not_exists => TRUE)",
			cfg.CompressionAfterHours,
		)); err != nil {
			return fmt.Errorf("add compression policy: %w", err)
		}
	}
	if cfg.RetentionDays > 0 {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(
			"SELECT add_retention_policy('market_bars', INTERVAL '%d days', if_not_exists => TRUE)",
			cfg.RetentionDays,
		)); err != nil {
			return fmt.Errorf("add retention policy: %w", err)
		}
	}
	return nil
}
