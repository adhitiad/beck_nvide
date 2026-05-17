# 🛠️ NVide Go Clean Architecture Module Scaffolder
# File: bEck_NVide/scaffold.ps1
# Cara penggunaan: .\scaffold.ps1 -Name "Post"

param (
    [Parameter(Mandatory=$true)]
    [string]$Name
)

# Validasi input
if ([string]::IsNullOrEmpty($Name)) {
    Write-Host "⚠️ Error: Nama modul tidak boleh kosong!" -ForegroundColor Red
    Exit
}

# Standarisasi Penamaan
# Contoh: "Post" -> CamelCase: "Post", lowercase: "post", plural: "posts"
$TitleName = $Name.Substring(0,1).ToUpper() + $Name.Substring(1)
$LowerName = $Name.ToLower()
$PluralName = $LowerName + "s"

Write-Host "🚀 Menghasilkan Boilerplate Clean Architecture untuk modul: $TitleName..." -ForegroundColor Cyan

# Path Tujuan
$DomainPath = "internal/domain/$LowerName.go"
$RepoPath = "internal/repository/$LowerName.go"
$UseCasePath = "internal/usecase/$LowerName.go"
$HandlerPath = "internal/delivery/${LowerName}_handlers.go"

# 1. GENERATE DOMAIN TEMPLATE
$DomainContent = @"
package domain

import (
	"context"
	"time"
)

// $TitleName mendefinisikan model data utama untuk entitas ini.
type $TitleName struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ${TitleName}Repository mendefinisikan kontrak data access (Database/Infrastructure layer).
type ${TitleName}Repository interface {
	Create(ctx context.Context, item *$TitleName) error
	GetByID(ctx context.Context, id UUID) (*$TitleName, error)
	List(ctx context.Context, limit, offset int) ([]*$TitleName, error)
	Update(ctx context.Context, item *$TitleName) error
	Delete(ctx context.Context, id UUID) error
}

// ${TitleName}UseCase mendefinisikan kontrak logika bisnis utama (Application layer).
type ${TitleName}UseCase interface {
	Create(ctx context.Context, userID UUID, title, content string) (*$TitleName, error)
	GetByID(ctx context.Context, id UUID) (*$TitleName, error)
	List(ctx context.Context, limit, offset int) ([]*$TitleName, error)
	Update(ctx context.Context, id UUID, userID UUID, title, content string) (*$TitleName, error)
	Delete(ctx context.Context, id UUID, userID UUID) error
}
"@

# 2. GENERATE REPOSITORY TEMPLATE
$RepoContent = @"
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type ${LowerName}Repository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// New${TitleName}Repository membuat instansiasi repository untuk $TitleName.
func New${TitleName}Repository(db *pgxpool.Pool, logger *zap.Logger) domain.${TitleName}Repository {
	return &${LowerName}Repository{
		db:     db,
		logger: logger,
	}
}

func (r *${LowerName}Repository) Create(ctx context.Context, item *domain.$TitleName) error {
	query := `
		INSERT INTO $PluralName (id, user_id, title, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query, item.ID, item.UserID, item.Title, item.Content)
	if err != nil {
		r.logger.Error("Failed to create $LowerName in DB", zap.Error(err))
		return err
	}
	return nil
}

func (r *${LowerName}Repository) GetByID(ctx context.Context, id domain.UUID) (*domain.$TitleName, error) {
	query := `
		SELECT id, user_id, title, content, created_at, updated_at
		FROM $PluralName
		WHERE id = $1
	`
	item := &domain.$TitleName{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.UserID,
		&item.Title,
		&item.Content,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewDomainError(domain.ErrCodeNotFound, "$LowerName not found", err)
		}
		r.logger.Error("Failed to get $LowerName by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}
	return item, nil
}

func (r *${LowerName}Repository) List(ctx context.Context, limit, offset int) ([]*domain.$TitleName, error) {
	query := `
		SELECT id, user_id, title, content, created_at, updated_at
		FROM $PluralName
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to query list of $PluralName", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	items := make([]*domain.$TitleName, 0)
	for rows.Next() {
		item := &domain.$TitleName{}
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Title,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *${LowerName}Repository) Update(ctx context.Context, item *domain.$TitleName) error {
	query := `
		UPDATE $PluralName
		SET title = $1, content = $2, updated_at = NOW()
		WHERE id = $3
	`
	tag, err := r.db.Exec(ctx, query, item.Title, item.Content, item.ID)
	if err != nil {
		r.logger.Error("Failed to update $LowerName in DB", zap.Error(err), zap.String("id", item.ID.String()))
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeNotFound, "$LowerName to update not found", nil)
	}
	return nil
}

func (r *${LowerName}Repository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM $PluralName WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete $LowerName from DB", zap.Error(err), zap.String("id", id.String()))
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeNotFound, "$LowerName to delete not found", nil)
	}
	return nil
}
"@

# 3. GENERATE USECASE TEMPLATE
$UseCaseContent = @"
package usecase

import (
	"context"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type ${LowerName}UseCase struct {
	${LowerName}Repo domain.${TitleName}Repository
	logger       *zap.Logger
}

// New${TitleName}UseCase membuat implementasi logika bisnis $TitleName.
func New${TitleName}UseCase(repo domain.${TitleName}Repository, logger *zap.Logger) domain.${TitleName}UseCase {
	return &${LowerName}UseCase{
		${LowerName}Repo: repo,
		logger:       logger,
	}
}

func (uc *${LowerName}UseCase) Create(ctx context.Context, userID domain.UUID, title, content string) (*domain.$TitleName, error) {
	if title == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "title is required", nil)
	}

	item := &domain.$TitleName{
		ID:      domain.NewUUID(),
		UserID:  userID,
		Title:   title,
		Content: content,
	}

	if err := uc.${LowerName}Repo.Create(ctx, item); err != nil {
		return nil, err
	}

	uc.logger.Info("$TitleName created successfully", zap.String("id", item.ID.String()), zap.String("user_id", userID.String()))
	return item, nil
}

func (uc *${LowerName}UseCase) GetByID(ctx context.Context, id domain.UUID) (*domain.$TitleName, error) {
	return uc.${LowerName}Repo.GetByID(ctx, id)
}

func (uc *${LowerName}UseCase) List(ctx context.Context, limit, offset int) ([]*domain.$TitleName, error) {
	if limit <= 0 {
		limit = 10
	}
	return uc.${LowerName}Repo.List(ctx, limit, offset)
}

func (uc *${LowerName}UseCase) Update(ctx context.Context, id domain.UUID, userID domain.UUID, title, content string) (*domain.$TitleName, error) {
	item, err := uc.${LowerName}Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi kepemilikan data (Owner checks)
	if item.UserID != userID {
		return nil, domain.NewDomainError(domain.ErrCodeForbidden, "you are not authorized to update this $LowerName", nil)
	}

	if title != "" {
		item.Title = title
	}
	if content != "" {
		item.Content = content
	}

	if err := uc.${LowerName}Repo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (uc *${LowerName}UseCase) Delete(ctx context.Context, id domain.UUID, userID domain.UUID) error {
	item, err := uc.${LowerName}Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validasi kepemilikan data (Owner checks)
	if item.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "you are not authorized to delete this $LowerName", nil)
	}

	return uc.${LowerName}Repo.Delete(ctx, id)
}
"@

# 4. GENERATE DELIVERY HANDLERS TEMPLATE
$HandlerContent = @"
package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// Catatan: Anda dapat mendaftarkan usecase baru ini di file internal/delivery/handlers.go 
// dengan menambahkannya ke struct Handler dan constructor NewHandler.

// Create$TitleName menangani pembuatan $TitleName baru
func (h *Handler) Create$TitleName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Memanggil usecase dinamis. Catatan: Tambahkan field ini ke struct Handler global.
	// Jika belum didaftarkan di global Handler, gunakan template casting / implementasi langsung.
	// Di bawah ini adalah contoh pemanggilan jika usecase sudah dipasang di handlers.go.
	/*
	item, err := h.${LowerName}UseCase.Create(r.Context(), userID, req.Title, req.Content)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, item)
	*/

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "Handler boilerplate untuk Create$TitleName terpanggil!",
		"mock_data": map[string]string{
			"user_id": userID.String(),
			"title":   req.Title,
			"content": req.Content,
		},
	})
}

// Get$TitleNameByID mengambil $TitleName berdasarkan ID
func (h *Handler) Get$TitleNameByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID format")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "Handler boilerplate untuk Get$TitleNameByID!",
		"id": id.String(),
	})
}

// List$PluralName mengambil daftar $TitleName berhalaman
func (h *Handler) List$PluralName(w http.ResponseWriter, r *http.Request) {
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "Handler boilerplate untuk List$PluralName!",
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
		},
	})
}
"@

# Membuat Direktori jika belum ada
New-Item -ItemType Directory -Force -Path "internal/domain" | Out-Null
New-Item -ItemType Directory -Force -Path "internal/repository" | Out-Null
New-Item -ItemType Directory -Force -Path "internal/usecase" | Out-Null
New-Item -ItemType Directory -Force -Path "internal/delivery" | Out-Null

# Menulis File Templates
$DomainContent | Out-File -FilePath $DomainPath -Encoding utf8
$RepoContent | Out-File -FilePath $RepoPath -Encoding utf8
$UseCaseContent | Out-File -FilePath $UseCasePath -Encoding utf8
$HandlerContent | Out-File -FilePath $HandlerPath -Encoding utf8

Write-Host "✅ Selesai! File boilerplate berikut telah berhasil dibuat:" -ForegroundColor Green
Write-Host " 📂 Domain:     $DomainPath" -ForegroundColor Gray
Write-Host " 📂 Repository: $RepoPath" -ForegroundColor Gray
Write-Host " 📂 Usecase:    $UseCasePath" -ForegroundColor Gray
Write-Host " 📂 Handlers:   $HandlerPath" -ForegroundColor Gray

Write-Host "`n💡 LANGKAH SELANJUTNYA UNTUK DEVELOPS:" -ForegroundColor Yellow
Write-Host "1. Daftarkan Repository dan Usecase baru Anda di [main.go](file:///E:/nvide/bEck_NVide/main.go):" -ForegroundColor Cyan
Write-Host "   -> ${LowerName}Repo := repository.New${TitleName}Repository(db.Pool(), logger)" -ForegroundColor Gray
Write-Host "   -> ${LowerName}UseCase := usecase.New${TitleName}UseCase(${LowerName}Repo, logger)" -ForegroundColor Gray
Write-Host "2. Tambahkan ${LowerName}UseCase ke struct 'Handler' di [internal/delivery/handlers.go](file:///E:/nvide/bEck_NVide/internal/delivery/handlers.go)" -ForegroundColor Cyan
Write-Host "3. Daftarkan endpoint REST baru Anda di [internal/delivery/router.go](file:///E:/nvide/bEck_NVide/internal/delivery/router.go):" -ForegroundColor Cyan
Write-Host "   -> protected.HandleFunc(\"/$PluralName\", handler.Create$TitleName).Methods(\"POST\")" -ForegroundColor Gray
Write-Host "   -> protected.HandleFunc(\"/$PluralName/{id}\", handler.Get$TitleNameByID).Methods(\"GET\")" -ForegroundColor Gray
Write-Host "   -> router.HandleFunc(\"/api/v1/$PluralName\", handler.List$PluralName).Methods(\"GET\")" -ForegroundColor Gray
