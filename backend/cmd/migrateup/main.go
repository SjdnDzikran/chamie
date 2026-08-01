package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/dzikran/chamie/internal/db"
)

func main() {
	migrationsPath := flag.String("path", "db/migrations", "path to migration files")
	down := flag.Bool("down", false, "roll back migrations instead of applying them")
	steps := flag.Int("steps", 1, "number of migrations to roll back; zero rolls back all")
	flag.Parse()

	_ = godotenv.Load()
	databaseURL := os.Getenv("DATABASE_URL")
	ctx := context.Background()

	if *down {
		if err := db.RunMigrationsDown(ctx, databaseURL, *migrationsPath, *steps); err != nil {
			log.Fatalf("migration rollback failed: %v", err)
		}
		fmt.Println("migrations rolled back")
		return
	}

	if err := db.RunMigrations(ctx, databaseURL, *migrationsPath); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied (or no change)")
}
