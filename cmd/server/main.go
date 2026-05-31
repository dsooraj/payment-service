package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"payment-service/internal/handler"
	"payment-service/internal/repository"
	"payment-service/internal/service"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func initDatabase() *sql.DB {

	connStr := os.Getenv("DB_URI")
	if connStr == "" {
		connStr = "postgres://appuser:secretpassword@localhost:5432/postgres?sslmode=disable"
	}

	var err error
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Error opening connection %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database %v", err)
	}

	query := `
		CREATE TABLE IF NOT EXISTS transactions (
			id SERIAL PRIMARY KEY,
			idempotency_key VARCHAR(255) UNIQUE NOT NULL,
			amount NUMERIC(10, 2) NOT NULL,
			currency VARCHAR(10) NOT NULL,
			status VARCHAR(20) NOT NULL,
			stripe_charge_id VARCHAR(255),
			created_At TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Error initializing table in database %v", err)
	}
	fmt.Println("Connected to database and established table")

	return db
}

func main() {

	db := initDatabase()
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to Payment Service")
	})

	fmt.Println("Server running in port 8080")

	txRepo := repository.NewPostgresTxRepository(db)
	service := service.NewPaymentService(txRepo)
	handler := handler.NewPaymentHandler(service)

	mux.HandleFunc("/payment", handler.HandleProcessPayment)

	wrappedMux := loggingMiddleware(mux)
	err := http.ListenAndServe(":8080", wrappedMux)
	if err != nil {
		fmt.Println("Error starting server", err)
	}
}

// To initialize: go mod init payment-service
// To execute: go run main.go

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		fmt.Printf("[API LOG] %s %s took %v", r.Method, r.URL, time.Since(start))
	})
}
