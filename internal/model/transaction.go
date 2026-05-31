package model

import "time"

type Transaction struct {
	Id             int       `json:"id"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	StripeChargeId string    `json:"stripe_charge_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}
