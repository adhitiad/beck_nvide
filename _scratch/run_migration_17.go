package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env not loaded")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("DATABASE_URL is required")
		return
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		return
	}
	defer conn.Close(ctx)

	fmt.Println("Connected to database. Reading migration script...")

	migrationFile := `migrations/017_high_privacy.sql`
	sqlContent, err := os.ReadFile(migrationFile)
	if err != nil {
		fmt.Printf("Failed to read migration file: %v\n", err)
		return
	}

	fmt.Println("Executing migration queries...")
	tag, err := conn.Exec(ctx, string(sqlContent))
	if err != nil {
		fmt.Printf("Failed to execute migration: %v\n", err)
		return
	}

	fmt.Printf("Migration executed successfully: %s\n", tag.String())
}
