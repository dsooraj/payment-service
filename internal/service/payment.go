package service

import (
	"errors"
	"fmt"
	"payment-service/internal/model"
	"payment-service/internal/repository"
)

type PaymentService struct {
	repo repository.TransactionRepository
}

func NewPaymentService(repo repository.TransactionRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (ps *PaymentService) ProcessPayment(idempotencyKey string, amount float64, currency string) (*model.Transaction, error) {
	existingTx, err := ps.repo.GetByIdempotencyKey(idempotencyKey)
	if err != nil {
		return nil, err
	}

	if existingTx != nil {
		return existingTx, nil
	}

	tx := &model.Transaction{
		IdempotencyKey: idempotencyKey,
		Amount:         amount,
		Currency:       currency,
		Status:         "pending",
	}

	if err = ps.repo.Create(tx); err != nil {
		return nil, fmt.Errorf("Failed to create a transaction %v", err)
	}

	if amount < 0 {
		tx.Status = "failed"
		ps.repo.UpdateStatus(tx.Id, tx.Status, "")
		return nil, errors.New("Invalid payment amount")
	}

	tx.Status = "completed"
	tx.StripeChargeId = fmt.Sprintf("stripe_id_%d", tx.Id)
	err = ps.repo.UpdateStatus(tx.Id, tx.Status, tx.StripeChargeId)

	if err != nil {
		return nil, fmt.Errorf("Failed to update status of transaction %d, %v", tx.Id, err)
	}

	return tx, nil

}
