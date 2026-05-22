package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	dbURL := "postgresql://postgres.rlzeronssvsefzgnpgjc:XvAyjeXH6Iu6zq1l@aws-1-us-east-1.pooler.supabase.com:5432/postgres"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	log.Println("Connected to database. Reading migration file...")

	migrationFile := "migrations/018_kyc_and_subscriptions.sql"
	sqlBytes, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migrationFile, err)
	}

	log.Println("Applying migrations...")
	_, err = conn.Exec(ctx, string(sqlBytes))
	if err != nil {
		log.Fatalf("Failed to execute migrations: %v", err)
	}

	fmt.Println("Migration successfully applied!")
}
