package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ksakiyama/study-cedar/internal/api"
	"github.com/ksakiyama/study-cedar/internal/cedar"
	_ "github.com/lib/pq"
)

func main() {
	// Get configuration from environment
	port := getEnv("PORT", "8080")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "cedardb")

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var db *sql.DB
	var err error

	// Retry connection for Docker startup timing
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database successfully")

	// Initialize Cedar authorizer
	authorizer, err := cedar.NewAuthorizer()
	if err != nil {
		log.Fatalf("Failed to initialize Cedar authorizer: %v", err)
	}
	log.Println("Cedar authorizer initialized successfully")

	// Create handler
	handler := api.NewHandler(db, authorizer)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Routes
	r.Get("/health", handler.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handler.HealthCheck)

		r.Route("/documents", func(r chi.Router) {
			r.Get("/", handler.ListDocuments)
			r.Post("/", handler.CreateDocument)
			r.Get("/{documentId}", handler.GetDocument)
			r.Put("/{documentId}", handler.UpdateDocument)
			r.Delete("/{documentId}", handler.DeleteDocument)
		})
	})

	// Create HTTP server
	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s", addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Setup signal handling for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal %v, starting graceful shutdown", sig)

		// Immediately mark as shutting down to fail health checks
		handler.SetShuttingDown(true)
		log.Println("Health check now returning 503")

		// Give existing connections time to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			if err := srv.Close(); err != nil {
				log.Printf("Force close failed: %v", err)
			}
		}

		log.Println("Server stopped gracefully")
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
