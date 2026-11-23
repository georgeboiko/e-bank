package repositories

import (
	"context"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *TransferRepository) List(ctx context.Context, ownerID *string, filter models.TransferListFilter) ([]models.Transfer, error) {
	if filter.AccountID != nil {
		if err := checkHistoryAccount(ctx, r.pool, *filter.AccountID, ownerID); err != nil {
			return nil, err
		}
	}

	userID := filter.UserID
	if ownerID != nil {
		userID = ownerID
	}
	rows, err := r.pool.Query(ctx, transferView+`
		WHERE ($1::text IS NULL OR sender.owner_id = $1 OR recipient.owner_id = $1)
		  AND ($2::integer IS NULL
		       OR ($3::text <> 'INCOMING' AND t.from_account_id = $2)
		       OR ($3 <> 'OUTGOING' AND t.to_account_id = $2))
		  AND ($4::timestamptz IS NULL OR t.created_at >= $4)
		  AND ($5::timestamptz IS NULL OR t.created_at < $5)
		  AND ($6::text = '' OR t.state = $6)
		  AND ($7::text = ''
		       OR ($7 = 'TOPUP' AND sender.type = 'TECHNICAL')
		       OR ($7 = 'TRANSFER' AND sender.type = 'CLIENT'))
		  AND ($8::numeric IS NULL OR t.amount >= $8)
		  AND ($9::numeric IS NULL OR t.amount <= $9)
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $10 OFFSET $11
	`, userID, filter.AccountID, filter.Direction, filter.From, filter.To, filter.State,
		filter.Type, filter.MinAmount, filter.MaxAmount, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Transfer])
}

func (r *AccountRepository) ListLedger(ctx context.Context, accountID int32, ownerID *string, filter models.HistoryFilter) ([]models.LedgerEntry, error) {
	if err := checkHistoryAccount(ctx, r.pool, accountID, ownerID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, transfer_id, account_id, amount, created_at
		FROM ledger_entry
		WHERE account_id = $1
		  AND ($2::timestamptz IS NULL OR created_at >= $2)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5
	`, accountID, filter.From, filter.To, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.LedgerEntry])
}

func checkHistoryAccount(ctx context.Context, pool *pgxpool.Pool, accountID int32, ownerID *string) error {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM account
			WHERE id = $1 AND ($2::text IS NULL OR (type = 'CLIENT' AND owner_id = $2))
		)
	`, accountID, ownerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return models.ErrAccountNotFound
	}
	return nil
}
