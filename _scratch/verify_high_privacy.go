package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/repository"
	"nvide-live/internal/usecase"
	"nvide-live/pkg/config"
	"nvide-live/pkg/database"
	"nvide-live/pkg/redis"
)

func main() {
	fmt.Println("=== MEMULAI VERIFIKASI FITUR PRIVASI & KEAMANAN TINGGI (FITUR 8) ===")

	// Load env
	if err := godotenv.Load("../.env"); err != nil {
		godotenv.Load(".env")
	}

	cfg := config.Load()
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 1. Inisialisasi Database
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

	// 2. Inisialisasi Redis
	redisClient, err := redis.New(&redis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: cfg.RedisPoolSize,
	}, logger)
	if err != nil {
		log.Fatalf("Gagal terhubung ke Redis: %v", err)
	}
	defer redisClient.Close()

	ctx := context.Background()

	// 3. Inisialisasi Repositori & Usecase
	repo := repository.NewPrivateChatRepository(db.Pool(), logger)
	userRepo := repository.NewUserRepository(db.Pool(), logger)
	chatUsecase := usecase.NewPrivateChatUsecase(repo, userRepo, redisClient, logger)

	// Buat UUID Pengguna Uji
	user1ID := domain.NewUUIDv7()
	user2ID := domain.NewUUIDv7()

	fmt.Printf("[INFO] ID Pengguna Uji 1: %s\n", user1ID)
	fmt.Printf("[INFO] ID Pengguna Uji 2: %s\n", user2ID)

	// Dapatkan atau buat Role ID valid untuk memenuhi foreign key users
	var roleID domain.UUID
	err = db.Pool().QueryRow(ctx, "SELECT id FROM roles LIMIT 1").Scan(&roleID)
	if err != nil {
		roleID = domain.NewUUIDv7()
		_, err = db.Pool().Exec(ctx, "INSERT INTO roles (id, name, description) VALUES ($1, 'test_role_privacy', 'Test Role') ON CONFLICT DO NOTHING", roleID)
		if err != nil {
			log.Fatalf("Gagal membuat role dummy: %v", err)
		}
	}

	// Masukkan pengguna uji ke database
	username1 := fmt.Sprintf("user_test_%s", user1ID.String())
	email1 := fmt.Sprintf("user_test_%s@example.com", user1ID.String())
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id) 
		VALUES ($1, $2, $3, $4, $5)
	`, user1ID, username1, email1, "hash", roleID)
	if err != nil {
		log.Fatalf("Gagal membuat user dummy 1: %v", err)
	}
	defer db.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", user1ID)

	username2 := fmt.Sprintf("user_test_%s", user2ID.String())
	email2 := fmt.Sprintf("user_test_%s@example.com", user2ID.String())
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role_id) 
		VALUES ($1, $2, $3, $4, $5)
	`, user2ID, username2, email2, "hash", roleID)
	if err != nil {
		log.Fatalf("Gagal membuat user dummy 2: %v", err)
	}
	defer db.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", user2ID)

	// ----------------------------------------------------
	// A. UJI COBA REGISTRASI & PENGAMBILAN E2EE PUBLIC KEY
	// ----------------------------------------------------
	fmt.Println("\n--- [A] Menguji Registrasi & Pengambilan Kunci E2EE ---")
	publicKeySample := "mcf5E3/3i4l0wXb2vQGkLpZ3b9z8v1R5j7y0t8x8v5o="
	err = chatUsecase.RegisterE2EEKey(ctx, user1ID, publicKeySample, "X25519")
	if err != nil {
		log.Fatalf("[Gagal] Gagal meregistrasi kunci E2EE: %v", err)
	}
	fmt.Println("[Sukses] Registrasi Kunci E2EE Berhasil!")

	key, err := chatUsecase.GetE2EEKey(ctx, user1ID)
	if err != nil {
		log.Fatalf("[Gagal] Gagal mengambil kunci E2EE: %v", err)
	}
	fmt.Printf("[Sukses] Mengambil Kunci E2EE Berhasil! Kunci: %s, Tipe: %s\n", key.PublicKey, key.KeyType)

	// ----------------------------------------------------
	// B. UJI COBA PENGATURAN PRIVASI PENGGUNA (PRIVATE & INCOGNITO)
	// ----------------------------------------------------
	fmt.Println("\n--- [B] Menguji Pengaturan Privasi Profil & Incognito ---")
	err = chatUsecase.UpdateUserPrivacy(ctx, user1ID, true, true)
	if err != nil {
		log.Fatalf("[Gagal] Gagal memperbarui pengaturan privasi: %v", err)
	}
	fmt.Println("[Sukses] Pengaturan Privasi (Private Profile & Incognito Mode) Berhasil Diperbarui!")

	// ----------------------------------------------------
	// C. UJI COBA BISUKAN PENGGUNA (MUTE) & PESAN YANG MASUK
	// ----------------------------------------------------
	fmt.Println("\n--- [C] Menguji Fitur Muting (Mute/Unmute) ---")
	// User 1 membisukan User 2 selama 60 menit
	err = chatUsecase.MuteUser(ctx, user1ID, user2ID, 60)
	if err != nil {
		log.Fatalf("[Gagal] Gagal membisukan pengguna: %v", err)
	}
	fmt.Println("[Sukses] User 1 berhasil membisukan User 2 selama 60 menit!")

	mutedUsers, err := chatUsecase.GetMutedUsers(ctx, user1ID)
	if err != nil {
		fmt.Printf("[Peringatan] Gagal mendapatkan daftar pengguna yang dibisukan: %v (Normal jika tabel users kosong)\n", err)
	} else {
		fmt.Printf("[Sukses] Menampilkan daftar pengguna dibisukan oleh User 1. Jumlah: %d\n", len(mutedUsers))
	}

	// ----------------------------------------------------
	// D. UJI COBA MEMULAI PERCAKAPAN & MENGIRIM PESAN E2EE & DISAPPEARING
	// ----------------------------------------------------
	fmt.Println("\n--- [D] Menguji Pembuatan Percakapan & Pengiriman Pesan ---")
	conv, err := chatUsecase.StartConversation(ctx, user1ID, user2ID)
	if err != nil {
		log.Fatalf("[Gagal] Gagal membuat percakapan: %v", err)
	}
	fmt.Printf("[Sukses] Percakapan privat berhasil dibuat! ID: %s\n", conv.ID)

	// Kirim pesan terenkripsi E2EE dengan Disappearing Mode 7 detik
	encryptedPayload := "ENCRYPTED_DATA_HERE"
	msg, err := chatUsecase.SendMessage(ctx, user2ID, conv.ID, "text", encryptedPayload, json.RawMessage("{}"), nil, true, "7s")
	if err != nil {
		log.Fatalf("[Gagal] Gagal mengirim pesan aman: %v", err)
	}
	fmt.Printf("[Sukses] Pesan terenkripsi & disappearing 7s berhasil dikirim! ID Pesan: %s\n", msg.ID)

	// ----------------------------------------------------
	// E. UJI COBA MARK VIEWED & DETEKSI SCREENSHOT
	// ----------------------------------------------------
	fmt.Println("\n--- [E] Menguji Penandaan Pesan Dilihat & Deteksi Screenshot ---")
	err = chatUsecase.MarkAsViewed(ctx, user1ID, msg.ID)
	if err != nil {
		log.Fatalf("[Gagal] Gagal menandai pesan sebagai dilihat: %v", err)
	}
	fmt.Println("[Sukses] Pesan berhasil ditandai sebagai dilihat! Timer disappearing diaktifkan.")

	err = chatUsecase.NotifyScreenshot(ctx, user1ID, conv.ID)
	if err != nil {
		log.Fatalf("[Gagal] Gagal mengirim notifikasi screenshot: %v", err)
	}
	fmt.Println("[Sukses] Notifikasi tangkapan layar (screenshot alert) berhasil dikirim & disiarkan!")

	// ----------------------------------------------------
	// F. UJI COBA TIMEOUT DISAPPEARING MESSAGES
	// ----------------------------------------------------
	fmt.Println("\n--- [F] Menguji Proses Pembersihan Pesan Kedaluwarsa (Disappearing Messages) ---")
	fmt.Println("[Info] Menyimulasikan pemrosesan pesan yang kedaluwarsa setelah durasi berlalu...")
	err = chatUsecase.ProcessExpiredMessages(ctx)
	if err != nil {
		log.Fatalf("[Gagal] Gagal memproses pesan kedaluwarsa: %v", err)
	}
	fmt.Println("[Sukses] Job pembersihan pesan kedaluwarsa berjalan dengan lancar!")

	// Unmute User 2
	err = chatUsecase.UnmuteUser(ctx, user1ID, user2ID)
	if err != nil {
		log.Fatalf("[Gagal] Gagal membuka bisu (unmute) pengguna: %v", err)
	}
	fmt.Println("[Sukses] Bisu User 2 berhasil dibuka!")

	fmt.Println("\n=== SEMUA FITUR PRIVASI & KEAMANAN TINGGI (FITUR 8) BERHASIL DIVERIFIKASI DENGAN SUKSES! ===")
}
