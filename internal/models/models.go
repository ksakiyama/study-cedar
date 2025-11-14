package models

import "time"

// Document represents a document in the system
type Document struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	OwnerID   string    `json:"owner_id" db:"owner_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DocumentInput represents input for creating/updating a document
type DocumentInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// DocumentsResponse represents a list of documents
type DocumentsResponse struct {
	Documents []Document `json:"documents"`
}
