package repository

import (
	"payment-service/internal/model"
)

type TransactionRepository interface {
	Create(tx *model.Transaction) error
	UpdateStatus(id int, status string, stripeId string) error
	GetByIdempotencyKey(key string) (*model.Transaction, error)
}
