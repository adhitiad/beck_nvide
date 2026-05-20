package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
)

type StorageHandler struct {
	uploadUC   *usecase.StorageUploadUseCase
	downloadUC *usecase.StorageDownloadUseCase
	logger     *zap.Logger
}

type StorageHandlerConfig struct {
	UploadUC   *usecase.StorageUploadUseCase
	DownloadUC *usecase.StorageDownloadUseCase
	Logger     *zap.Logger
}

func NewStorageHandler(cfg *StorageHandlerConfig) *StorageHandler {
	return &StorageHandler{
		uploadUC:   cfg.UploadUC,
		downloadUC: cfg.DownloadUC,
		logger:     cfg.Logger,
	}
}

func (h *StorageHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/storage/upload", h.Upload).Methods("POST")
	r.HandleFunc("/storage/{id}/url", h.GetPresignedURL).Methods("GET")
	r.HandleFunc("/storage/{id}/urls", h.GetPresignedURLs).Methods("GET")
}

func (h *StorageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	contentTypeStr := r.URL.Query().Get("content_type")
	if contentTypeStr == "" {
		contentTypeStr = "backup"
	}

	contentTypeEnum := domain.ContentType(contentTypeStr)

	key := h.uploadUC.GenerateKey()
	objectName := header.Filename
	if objectName != "" {
		key = key + "_" + objectName
	}

	storageFile, err := h.uploadUC.UploadWithReader(r.Context(), "default-bucket", key, file, contentTypeEnum, nil)
	if err != nil {
		h.logger.Error("Upload failed", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, storageFile)
}

func (h *StorageHandler) GetPresignedURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	if fileID == "" {
		h.respondError(w, http.StatusBadRequest, "file id is required")
		return
	}

	storageFile, err := h.downloadUC.GetPresignedURL(r.Context(), domain.UUID(fileID))
	if err != nil {
		h.logger.Error("Get presigned URL failed", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"url": storageFile})
}

func (h *StorageHandler) GetPresignedURLs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	if fileID == "" {
		h.respondError(w, http.StatusBadRequest, "file id is required")
		return
	}

	urls, err := h.downloadUC.GetPresignedURLs(r.Context(), domain.UUID(fileID))
	if err != nil {
		h.logger.Error("Get presigned URLs failed", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, urls)
}

func (h *StorageHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *StorageHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}