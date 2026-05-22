package scratch

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"nvide-live/internal/domain"
	"nvide-live/internal/repository"
	"go.uber.org/zap"
)

func CheckAuth() {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/nvide_live?sslmode=disable"
	}

	fmt.Println("Connecting to DB:", databaseURL)
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Create logger
	logger, _ := zap.NewDevelopment()

	// 1. Find a user in the database
	var idStr string
	var email string
	err = pool.QueryRow(context.Background(), "SELECT id, email FROM users LIMIT 1").Scan(&idStr, &email)
	if err != nil {
		log.Fatalf("Failed to fetch a user: %v\n", err)
	}
	fmt.Printf("Found user in DB: ID=%s, Email=%s\n", idStr, email)

	// 2. Call GetByID
	userRepo := repository.NewUserRepository(pool, logger)
	userID, err := domain.FromString(idStr)
	if err != nil {
		log.Fatalf("Invalid domain UUID: %v\n", err)
	}

	fmt.Printf("Calling GetByID with UUID: %s...\n", userID)
	user, err := userRepo.GetByID(context.Background(), userID)
	if err != nil {
		fmt.Printf("GetByID FAILED with error: %v\n", err)
	} else {
		fmt.Printf("GetByID SUCCESS! User: %+v\n", user)
	}
}
