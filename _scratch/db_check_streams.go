package scratch

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func CheckStreams() {
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

	// Print columns of gift_fraud_alerts
	rows, err := conn.Query(context.Background(), 
		"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'gift_fraud_alerts' ORDER BY column_name")
	if err != nil {
		log.Fatalf("Query columns failed: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("\nColumns in 'gift_fraud_alerts' table:")
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("- %s: %s\n", columnName, dataType)
	}
}
