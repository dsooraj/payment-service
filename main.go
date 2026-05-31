package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PaymentRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Method   string  `json:"method"`
}

func (pr PaymentRequest) Validate() error {
	if pr.Amount == 0 || pr.Amount < 0 {
		return errors.New("Amount should be positive")
	}

	if pr.Currency == "" {
		return errors.New("Currency is required")
	}

	if pr.Method == "" {
		return errors.New("Method is required")
	}

	return nil
}

type Transaction struct {
	Id       int     `json:"id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Method   string  `json:"method"`
	Status   string  `json:"status"`
}

type PaymentProcessor interface {
	Process(amount float64, currency string) (string, error)
}

type MockProcessor struct{}

func (mp *MockProcessor) Process(amount float64, currency string) (string, error) {

	time.Sleep(500 * time.Millisecond)

	if amount < 0 {
		return "", errors.New("Amount is insufficient")
	}

	txnId := time.Now().UnixNano()
	return fmt.Sprintf("txn_gateway_id_%d", txnId), nil
}

var (
	db      *sql.DB
	gateway PaymentProcessor = &MockProcessor{}
)

func initDatabase() {
	connStr := "postgres://appuser:secretpassword@localhost:5432/postgres?sslmode=disable"

	var err error
	db, err = sql.Open("pgx", connStr)
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
			amount NUMERIC(10, 2),
			currency VARCHAR(10),
			method VARCHAR(50),
			status VARCHAR(20)
		);
	`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Error initializing table in database %v", err)
	}
	fmt.Println("Connected to database and established table")
}

func main() {

	initDatabase()
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to Payment Service")
	})

	fmt.Println("Server running in port 8080")

	mux.HandleFunc("/payment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PaymentRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Incorrect body", http.StatusBadGateway)
			return
		}

		if err = req.Validate(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"message": err.Error(),
			})
			return
		}

		_, gatewayErr := gateway.Process(req.Amount, req.Currency)

		status := "completed"
		if gatewayErr != nil {
			status = "failed"
		}

		query := `
			INSERT INTO transactions (amount, currency, method, status)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`

		var newId int
		dbErr := db.QueryRow(query, req.Amount, req.Currency, req.Method, status).Scan(&newId)
		if dbErr != nil {
			fmt.Printf("Error adding data to database %v", dbErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tx := Transaction{
			Id:       newId,
			Amount:   req.Amount,
			Currency: req.Currency,
			Method:   req.Method,
			Status:   status,
		}

		fmt.Printf("Processed payment %.2f %s %s. Status %s\n", req.Amount, req.Currency, req.Method, status)
		fmt.Printf("Transaction Id %d.\n", newId)

		w.Header().Set("content-type", "application/json")

		if status == "failed" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{
				"message": gatewayErr.Error(),
				"status":  status,
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tx)
	})

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
