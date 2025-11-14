package models

import (
	"database/sql"
	"time"
)

// Document represents a document in the system
type Document struct {
	ID              string         `json:"id" db:"id"`
	Title           string         `json:"title" db:"title"`
	Content         string         `json:"content" db:"content"`
	OwnerID         string         `json:"owner_id" db:"owner_id"`
	DocumentGroupID sql.NullString `json:"document_group_id,omitempty" db:"document_group_id"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

// UserGroup represents a user group in the system
type UserGroup struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// DocumentGroup represents a document group in the system
type DocumentGroup struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GroupAssociation represents the N:N relationship between document groups and user groups
type GroupAssociation struct {
	ID              int       `json:"id" db:"id"`
	DocumentGroupID string    `json:"document_group_id" db:"document_group_id"`
	UserGroupID     string    `json:"user_group_id" db:"user_group_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
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
