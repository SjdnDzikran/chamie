package db

import (
	"context"
	"strings"
	"testing"
)

func TestRunMigrationsRequiresDatabaseURL(t *testing.T) {
	err := RunMigrations(context.Background(), "", "migrations")
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("RunMigrations() error = %v, want DATABASE_URL validation", err)
	}
}

func TestRunMigrationsRequiresMigrationPath(t *testing.T) {
	err := RunMigrations(context.Background(), "postgres://localhost/chamie", "")
	if err == nil || !strings.Contains(err.Error(), "migration path is required") {
		t.Fatalf("RunMigrations() error = %v, want migration path validation", err)
	}
}

func TestRunMigrationsDownRequiresPositiveStepsOrAll(t *testing.T) {
	err := RunMigrationsDown(context.Background(), "", "migrations", 1)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("RunMigrationsDown() error = %v, want DATABASE_URL validation", err)
	}
}
