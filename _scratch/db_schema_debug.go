package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"nvide-live/pkg/config"
	"nvide-live/pkg/database"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		godotenv.Load(".env")
	}

	cfg := config.Load()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	db, err := database.New(&database.Config{
		DATABASE_URL: cfg.DATABASE_URL,
		Host:         cfg.DBHost,
		Port:         cfg.DBPort,
		User:         cfg.DBUser,
		Password:     cfg.DBPassword,
		DBName:       cfg.DBName,
		SSLMode:      cfg.DBSSLMode,
		MaxConn:      cfg.DBMaxConn,
		MinConn:      cfg.DBMinConn,
	}, logger)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Query columns of 'public.messages'
	fmt.Println("=== COLUMNS IN TABLE 'public.messages' ===")
	rows, err := db.Pool().Query(ctx, `
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'messages' AND table_schema = 'public'
	`)
	if err != nil {
		log.Fatalf("Gagal query columns: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, isNullable string
		if err := rows.Scan(&colName, &dataType, &isNullable); err != nil {
			log.Fatalf("Gagal scan column info: %v", err)
		}
		fmt.Printf("Column: %s, Type: %s, Nullable: %s\n", colName, dataType, isNullable)
	}

	// Query tables list
	fmt.Println("\n=== ALL TABLES ===")
	tRows, err := db.Pool().Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
	`)
	if err != nil {
		log.Fatalf("Gagal query tables: %v", err)
	}
	defer tRows.Close()

	for tRows.Next() {
		var tblName string
		if err := tRows.Scan(&tblName); err != nil {
			log.Fatalf("Gagal scan table info: %v", err)
		}
		fmt.Println("Table:", tblName)
	}
}
