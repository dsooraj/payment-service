package repository

import (
	"database/sql"
	"payment-service/internal/model"
)

type PostgresTxRepository struct {
	DB *sql.DB
}

func NewPostgresTxRepository(db *sql.DB) *PostgresTxRepository {
	return &PostgresTxRepository{DB: db}
}

func (ptx *PostgresTxRepository) Create(tx *model.Transaction) error {
	query := `
		INSERT INTO transactions (idempotency_key, amount, currency, status)
		VALUES ($1,$2,$3,$4)
		RETURNING id, created_at;
	`
	return ptx.DB.QueryRow(query, tx.IdempotencyKey, tx.Amount, tx.Currency, tx.Status).Scan(&tx.Id, &tx.CreatedAt)
}

func (ptx *PostgresTxRepository) GetByIdempotencyKey(key string) (*model.Transaction, error) {
	query := `
		SELECT id, idempotency_key, amount, currency, status , created_at
		FROM transactions
		WHERE idempotency_key = $1;
	`

	var tx *model.Transaction
	err := ptx.DB.QueryRow(query, key).Scan(&tx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (ptx *PostgresTxRepository) UpdateStatus(id int, status string, stripeId string) error {
	query := `
		UPDATE transactions
		SET status = $1, stripe_charge_id = $2
		WHERE id = $3;
	`

	_, err := ptx.DB.Exec(query, status, stripeId, id)

	if err != nil {
		return err
	}

	return nil
}
