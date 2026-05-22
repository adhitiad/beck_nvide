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

	fmt.Println("=== MEMPERBAIKI SKEMA TABEL 'messages' DI DATABASE ===")

	// Drop NOT NULL or drop columns room_id and user_id from messages table
	_, err = db.Pool().Exec(ctx, `
		ALTER TABLE messages ALTER COLUMN room_id DROP NOT NULL;
		ALTER TABLE messages ALTER COLUMN user_id DROP NOT NULL;
	`)
	if err != nil {
		log.Fatalf("Gagal mengubah kolom room_id dan user_id di tabel messages: %v", err)
	}

	fmt.Println("[Sukses] Kolom room_id dan user_id di tabel messages berhasil diubah menjadi NULLABLE!")
}
