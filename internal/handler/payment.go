package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"payment-service/internal/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(s *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: s}
}

type PaymentRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

func (pr PaymentRequest) Validate() error {
	if pr.Amount == 0 || pr.Amount < 0 {
		return errors.New("Amount should be positive")
	}

	if pr.Currency == "" {
		return errors.New("Currency is required")
	}

	return nil
}

func (h *PaymentHandler) HandleProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idempotencyKey := r.Header.Get("idempotency-key")
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

	tx, processErr := h.service.ProcessPayment(idempotencyKey, req.Amount, req.Currency)

	w.Header().Set("content-type", "application/json")

	if tx.Status == "failed" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"message": processErr.Error(),
			"status":  tx.Status,
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tx)
}
