package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type PaymentRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Method   string  `json:"method"`
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
	transactionDb = make(map[int]Transaction)
	nextId        = 1
	mu            sync.Mutex
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to Payment Service")
	})

	fmt.Println("Server running in port 8080")

	http.HandleFunc("/payment", func(w http.ResponseWriter, r *http.Request) {
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

		mu.Lock()
		defer mu.Unlock()
		tx := Transaction{
			Id:       nextId,
			Amount:   req.Amount,
			Currency: req.Currency,
			Method:   req.Method,
			Status:   "completed",
		}

		transactionDb[nextId] = tx
		nextId++

		processer := MockProcessor{}
		_, gatewayErr := processer.Process(req.Amount, req.Currency)

		status := "completed"
		if gatewayErr != nil {
			status = "failed"
		}

		fmt.Printf("Processed payment %.2f %s %s. Status %s\n", req.Amount, req.Currency, req.Method, status)
		fmt.Printf("Transaction Id %d. Total transactions: %d\n", nextId-1, len(transactionDb))

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
		// fmt.Fprint(w, "Payment received")

	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server", err)
	}
}

// To initialize: go mod init payment-service
// To execute: go run main.go
