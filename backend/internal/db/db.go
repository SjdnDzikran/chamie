package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func RunMigrations(ctx context.Context, databaseURL, migrationsPath string) error {
	migrator, err := newMigrator(ctx, databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrator(migrator)

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func RunMigrationsDown(ctx context.Context, databaseURL, migrationsPath string, steps int) error {
	migrator, err := newMigrator(ctx, databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrator(migrator)

	if steps <= 0 {
		err = migrator.Down()
	} else {
		err = migrator.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("rollback migrations: %w", err)
	}
	return nil
}

func newMigrator(ctx context.Context, databaseURL, migrationsPath string) (*migrate.Migrate, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(migrationsPath) == "" {
		return nil, fmt.Errorf("migration path is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("resolve migration path: %w", err)
	}
	sourceURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}

	driver, err := migratepgx.WithInstance(sqlDB, &migratepgx.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create migration driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(sourceURL, "pgx5", driver)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return migrator, nil
}

func closeMigrator(migrator *migrate.Migrate) {
	if migrator == nil {
		return
	}
	_, _ = migrator.Close()
}
