package scratch

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func CheckDB() {
	// Load .env
	_ = godotenv.Load("../.env")
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/nvide_live?sslmode=disable"
	}

	fmt.Println("Connecting to:", databaseURL)
	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	// 1. Check if table users exists
	var exists bool
	err = conn.QueryRow(context.Background(), 
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users')").Scan(&exists)
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	fmt.Printf("Table 'users' exists: %v\n", exists)

	if !exists {
		return
	}

	// 2. Print columns
	rows, err := conn.Query(context.Background(), 
		"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users' ORDER BY column_name")
	if err != nil {
		log.Fatalf("Query columns failed: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("\nColumns in 'users' table:")
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s: %s\n", columnName, dataType)
	}
}
