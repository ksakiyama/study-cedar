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
	"github.com/ksakiyama/study-cedar/internal/iputil"
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

// checkGroupAccess checks if a user group has access to a document group
func (h *Handler) checkGroupAccess(userGroupID, documentGroupID string) bool {
	// If either group ID is empty, no group access
	if userGroupID == "" || documentGroupID == "" {
		return false
	}

	// Check if association exists
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM group_associations
			WHERE user_group_id = $1 AND document_group_id = $2
		)
	`, userGroupID, documentGroupID).Scan(&exists)

	if err != nil {
		return false
	}

	return exists
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
	userGroupID := r.Header.Get("X-User-Group-ID")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Get IP address information
	ipInfo := iputil.GetIPInfo(r)

	// Check basic authorization (for listing, we set has_group_access to true for role-based check)
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:         userID,
		UserRole:       userRole,
		Action:         "ListDocuments",
		ResourceID:     "documents",
		IPAddress:      ipInfo.IPAddress,
		IsPrivateIP:    ipInfo.IsPrivateIP,
		IsJapanIP:      ipInfo.IsJapanIP,
		HasGroupAccess: true, // For list operation, check role only
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied: Geographic restriction or insufficient permissions")
		return
	}

	// Fetch documents from database with group filtering
	var rows *sql.Rows
	if userRole == "admin" {
		// Admins can see all documents
		rows, err = h.db.Query(`
			SELECT id, title, content, owner_id, document_group_id, created_at, updated_at
			FROM documents
			ORDER BY created_at DESC
		`)
	} else if userGroupID != "" {
		// Users with group: only show documents from associated groups
		rows, err = h.db.Query(`
			SELECT d.id, d.title, d.content, d.owner_id, d.document_group_id, d.created_at, d.updated_at
			FROM documents d
			LEFT JOIN group_associations ga ON d.document_group_id = ga.document_group_id
			WHERE ga.user_group_id = $1 OR d.document_group_id IS NULL
			ORDER BY d.created_at DESC
		`, userGroupID)
	} else {
		// Users without group: only show documents without group
		rows, err = h.db.Query(`
			SELECT id, title, content, owner_id, document_group_id, created_at, updated_at
			FROM documents
			WHERE document_group_id IS NULL
			ORDER BY created_at DESC
		`)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}
	defer rows.Close()

	documents := []models.Document{}
	for rows.Next() {
		var doc models.Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.DocumentGroupID, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
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
	userGroupID := r.Header.Get("X-User-Group-ID")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner and group
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, title, content, owner_id, document_group_id, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.DocumentGroupID, &doc.CreatedAt, &doc.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check group access
	hasGroupAccess := false
	if doc.DocumentGroupID.Valid {
		hasGroupAccess = h.checkGroupAccess(userGroupID, doc.DocumentGroupID.String)
	} else {
		// Document has no group, allow access
		hasGroupAccess = true
	}

	// Get IP address information
	ipInfo := iputil.GetIPInfo(r)

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "GetDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
		IPAddress:       ipInfo.IPAddress,
		IsPrivateIP:     ipInfo.IsPrivateIP,
		IsJapanIP:       ipInfo.IsJapanIP,
		HasGroupAccess:  hasGroupAccess,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied: Geographic restriction or insufficient permissions")
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

	// Get IP address information
	ipInfo := iputil.GetIPInfo(r)

	// Check authorization (for creation, use role-based access only)
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:         userID,
		UserRole:       userRole,
		Action:         "CreateDocument",
		ResourceID:     "documents",
		IPAddress:      ipInfo.IPAddress,
		IsPrivateIP:    ipInfo.IsPrivateIP,
		IsJapanIP:      ipInfo.IsJapanIP,
		HasGroupAccess: true, // For creation, check role only
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied: Geographic restriction or insufficient permissions")
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
	userGroupID := r.Header.Get("X-User-Group-ID")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner and group
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, title, content, owner_id, document_group_id, created_at, updated_at
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.Title, &doc.Content, &doc.OwnerID, &doc.DocumentGroupID, &doc.CreatedAt, &doc.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check group access
	hasGroupAccess := false
	if doc.DocumentGroupID.Valid {
		hasGroupAccess = h.checkGroupAccess(userGroupID, doc.DocumentGroupID.String)
	} else {
		hasGroupAccess = true
	}

	// Get IP address information
	ipInfo := iputil.GetIPInfo(r)

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "UpdateDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
		IPAddress:       ipInfo.IPAddress,
		IsPrivateIP:     ipInfo.IsPrivateIP,
		IsJapanIP:       ipInfo.IsJapanIP,
		HasGroupAccess:  hasGroupAccess,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied: Geographic restriction or insufficient permissions")
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
	userGroupID := r.Header.Get("X-User-Group-ID")

	if userID == "" || userRole == "" {
		respondError(w, http.StatusBadRequest, "Missing user headers")
		return
	}

	// Fetch document to get owner and group
	var doc models.Document
	err := h.db.QueryRow(`
		SELECT id, owner_id, document_group_id
		FROM documents
		WHERE id = $1
	`, documentID).Scan(&doc.ID, &doc.OwnerID, &doc.DocumentGroupID)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Document not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Database error: %v", err))
		return
	}

	// Check group access
	hasGroupAccess := false
	if doc.DocumentGroupID.Valid {
		hasGroupAccess = h.checkGroupAccess(userGroupID, doc.DocumentGroupID.String)
	} else {
		hasGroupAccess = true
	}

	// Get IP address information
	ipInfo := iputil.GetIPInfo(r)

	// Check authorization
	authorized, err := h.authorizer.Authorize(cedar.AuthzRequest{
		UserID:          userID,
		UserRole:        userRole,
		Action:          "DeleteDocument",
		ResourceID:      documentID,
		ResourceOwnerID: doc.OwnerID,
		IPAddress:       ipInfo.IPAddress,
		IsPrivateIP:     ipInfo.IsPrivateIP,
		IsJapanIP:       ipInfo.IsJapanIP,
		HasGroupAccess:  hasGroupAccess,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Authorization error: %v", err))
		return
	}

	if !authorized {
		respondError(w, http.StatusForbidden, "Access denied: Geographic restriction or insufficient permissions")
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
