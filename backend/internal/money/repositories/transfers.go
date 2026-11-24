package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const transferView = `
		SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.state, t.reject_cause,
		       t.created_at, t.updated_at, t.finished_at,
		       CASE sender.type WHEN 'TECHNICAL' THEN 'TOPUP' ELSE 'TRANSFER' END AS type
		FROM transfer AS t
		JOIN account AS sender ON sender.id = t.from_account_id
		JOIN account AS recipient ON recipient.id = t.to_account_id
`

type TransferRepository struct {
	pool *pgxpool.Pool
}

func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{pool: pool}
}

func (r *TransferRepository) Create(ctx context.Context, input models.CreateTransfer) (models.Transfer, bool, error) {
	transfer := models.Transfer{Type: "TRANSFER"}
	var created bool
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		var transferID int32
		var eventID pgtype.UUID
		err := tx.QueryRow(ctx, `
			SELECT create_transfer($1, $2, $3, $4, $5::text::uuid, event_id, $6, $7), event_id
			FROM (SELECT uuidv7() AS event_id) AS candidate
		`, input.ActorID, input.FromAccountID, input.ToAccountID, input.Amount, input.IdempotencyKey,
			input.DeviceID, input.IPAddress).Scan(&transferID, &eventID)
		if err != nil {
			return transferCreateError(err)
		}

		// Only a new transfer uses the generated event ID.
		return tx.QueryRow(ctx, `
			SELECT id, from_account_id, to_account_id, amount, state, reject_cause,
			       created_at, updated_at, finished_at,
			       EXISTS (SELECT 1 FROM outbox_event WHERE transfer_id = t.id AND event_id = $2)
			FROM transfer AS t WHERE id = $1
		`, transferID, eventID).Scan(
			&transfer.ID, &transfer.FromAccountID, &transfer.ToAccountID, &transfer.Amount,
			&transfer.State, &transfer.RejectCause, &transfer.CreatedAt, &transfer.UpdatedAt,
			&transfer.FinishedAt, &created,
		)
	})
	if err != nil {
		return models.Transfer{}, false, err
	}
	return transfer, created, nil
}

func (r *TransferRepository) Get(ctx context.Context, id int32, ownerID *string) (models.Transfer, error) {
	rows, err := r.pool.Query(ctx, transferView+`
		WHERE t.id = $1
		  AND ($2::text IS NULL OR sender.owner_id = $2 OR recipient.owner_id = $2)
	`, id, ownerID)
	if err != nil {
		return models.Transfer{}, err
	}
	transfer, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Transfer])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Transfer{}, models.ErrTransferNotFound
	}
	return transfer, err
}

func transferCreateError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "22023":
		return models.ErrInvalidTransfer
	case "P0002":
		return models.ErrAccountNotFound
	case "P3311":
		return models.ErrIdempotencyConflict
	case "P3312":
		return models.ErrInactiveAccount
	case "P3313":
		return models.ErrInsufficientFunds
	case "P3314":
		return models.ErrTransferLimit
	case "P3317":
		return models.ErrLimitsNotConfigured
	default:
		return err
	}
}
