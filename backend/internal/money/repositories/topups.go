package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type TopupRepository struct {
	pool *pgxpool.Pool
}

func NewTopupRepository(pool *pgxpool.Pool) *TopupRepository {
	return &TopupRepository{pool: pool}
}

func (r *TopupRepository) Create(ctx context.Context, technicalID int32, input models.CreateTopup) (models.Topup, bool, error) {
	var topup models.Topup
	var exists bool
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		// Serialize requests with the same key while determining the HTTP creation status.
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended('money.create_topup:' || (($1::text)::uuid)::text, 0))
		`, input.IdempotencyKey); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM transfer WHERE idempotency_key = $1::text::uuid)
		`, input.IdempotencyKey).Scan(&exists)
		if err != nil {
			return err
		}
		var id int32
		err = tx.QueryRow(ctx, `SELECT create_topup($1, $2, $3, $4, $5::text::uuid)`,
			input.ActorID, technicalID, input.AccountID, input.Amount, input.IdempotencyKey).Scan(&id)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "transfer_idempotency_key_key" {
				return models.ErrIdempotencyConflict
			}
			return transferCreateError(err)
		}
		topup, err = readTopup(ctx, tx, id, &input.ActorID)
		return err
	})
	if err != nil {
		return models.Topup{}, false, err
	}
	return topup, !exists, nil
}

func (r *TopupRepository) Get(ctx context.Context, id int32, owner *string) (models.Topup, error) {
	return readTopup(ctx, r.pool, id, owner)
}

type topupReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readTopup(ctx context.Context, db topupReader, id int32, owner *string) (models.Topup, error) {
	var topup models.Topup
	err := db.QueryRow(ctx, `
		SELECT t.id, t.to_account_id, t.amount, t.state, e.external_payment_id, t.idempotency_key::text,
		       t.created_at, t.updated_at, t.finished_at
		FROM transfer t
		JOIN account sender ON sender.id = t.from_account_id AND sender.type = 'TECHNICAL'
		JOIN account recipient ON recipient.id = t.to_account_id AND recipient.type = 'CLIENT'
		LEFT JOIN external_payment e ON e.transfer_id = t.id
		WHERE t.id = $1 AND ($2::text IS NULL OR recipient.owner_id = $2)
	`, id, owner).Scan(&topup.ID, &topup.AccountID, &topup.Amount, &topup.State, &topup.PaymentID,
		&topup.IdempotencyKey, &topup.CreatedAt, &topup.UpdatedAt, &topup.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Topup{}, models.ErrTopupNotFound
	}
	return topup, err
}

func (r *TopupRepository) RegisterPayment(ctx context.Context, topupID int32, paymentID string) error {
	_, err := r.pool.Exec(ctx, `SELECT register_topup_payment($1, $2)`, topupID, paymentID)
	return err
}

func (r *TopupRepository) ConfirmPayment(ctx context.Context, topupID int32, paymentID string, amount decimal.Decimal, currency string) error {
	_, err := r.pool.Exec(ctx, `SELECT confirm_topup($1, $2, $3, $4)`, topupID, paymentID, amount, currency)
	return topupUpdateError(err)
}

func (r *TopupRepository) CancelPayment(ctx context.Context, topupID int32, paymentID string) error {
	_, err := r.pool.Exec(ctx, `SELECT cancel_topup($1, $2)`, topupID, paymentID)
	return topupUpdateError(err)
}

func topupUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "P0002":
		return models.ErrTopupNotFound
	case "P3312":
		return models.ErrInactiveAccount
	case "P3315":
		return models.ErrTopupPaymentMismatch
	case "P3316":
		return models.ErrTopupInvalidState
	default:
		return err
	}
}
