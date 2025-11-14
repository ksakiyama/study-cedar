package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ksakiyama/study-cedar/internal/cedar"
	"github.com/ksakiyama/study-cedar/internal/models"
)

// Handler contains dependencies for API handlers
type Handler struct {
	db            *sql.DB
	authorizer    *cedar.Authorizer
	isShuttingDown atomic.Bool
}

// NewHandler creates a new API handler
func NewHandler(db *sql.DB, authorizer *cedar.Authorizer) *Handler {
	return &Handler{
		db:         db,
		authorizer: authorizer,
	}
}

// SetShuttingDown sets the shutting down state
func (h *Handler) SetShuttingDown(shuttingDown bool) {
	h.isShuttingDown.Store(shuttingDown)
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// If server is shutting down, return 503 Service Unavailable
	if h.isShuttingDown.Load() {
		response := models.HealthResponse{
			Status: "shutting_down",
		}
		respondJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	response := models.HealthResponse{
		Status: "ok",
	}
	respondJSON(w, http.StatusOK, response)
}

// ListDocuments handles document listing
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:     userID,
		UserRole:   userRole,
		Action:     "ListDocuments",
		ResourceID: "documents",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Fetch documents from database
	rows, err := h.db.Query(`
		SELECT id, title, content, owner_id, created_at, updated_at
		FROM documents
		ORDER BY created_at DESC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}
	defer rows.Close()

	documents := []models.Document{}
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Scan error: %v", err))
			return
		}
		documents = append(documents, doc)
	}

	response := models.DocumentsResponse{
		Documents: documents,
	}
	respondJSON(w, http.StatusOK, response)
}

// GetDocument handles fetching a single document
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, title, content, owner_id, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "GetDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// CreateDocument handles document creation
func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:     userID,
		UserRole:   userRole,
		Action:     "CreateDocument",
		ResourceID: "documents",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse request body
	var input models.DocumentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create document
	doc := models.Document{
		ID:        fmt.Sprintf("doc-%d", time.Now().Unix()),
		Title:     input.Title,
		Content:   input.Content,
		OwnerID:   userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = h.db.Exec(`
		INSERT INTO documents (id, title, content, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, doc.ID, doc.Title, doc.Content, doc.OwnerID, doc.CreatedAt, doc.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

// UpdateDocument handles document updates
func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, title, content, owner_id, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "UpdateDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse request body
	var input models.DocumentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update document
	doc.Title = input.Title
	doc.Content = input.Content
	doc.UpdatedAt = time.Now()

	_, err = h.db.Exec(`
		UPDATE documents
		SET title = $1, content = $2, updated_at = $3
		WHERE id = $4
	`, doc.Title, doc.Content, doc.UpdatedAt, doc.ID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// DeleteDocument handles document deletion
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	userID := r.Header.Get("X-User-ID")
	userRole := r.Header.Get("X-User-Role")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, owner_id
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.OwnerID)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "DeleteDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete document
	_, err = h.db.Exec(`DELETE FROM documents WHERE id = $1`, documentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, models.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
