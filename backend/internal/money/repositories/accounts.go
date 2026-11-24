package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const accountColumns = `id, owner_id, type, state, balance, reserved_sum,
	(balance - reserved_sum) AS available_balance, opened_at`

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) ListAccounts(ctx context.Context, ownerID *string, state string, limit int, offset int64) ([]models.Account, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+accountColumns+`
		FROM account
		WHERE ($1::text IS NULL OR (type = 'CLIENT' AND owner_id = $1))
		  AND ($2::text = '' OR state = $2)
		ORDER BY opened_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, ownerID, state, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Account])
}

func (r *AccountRepository) GetAccount(ctx context.Context, id int32, ownerID *string) (models.Account, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+accountColumns+`
		FROM account
		WHERE id = $1
		  AND ($2::text IS NULL OR (type = 'CLIENT' AND owner_id = $2))
	`, id, ownerID)
	if err != nil {
		return models.Account{}, err
	}
	account, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Account])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Account{}, models.ErrAccountNotFound
	}
	return account, err
}

func (r *AccountRepository) SetState(
	ctx context.Context,
	id int32,
	state, actorID string,
	transition func(models.Account) (string, error),
) (models.Account, error) {
	var account models.Account
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+accountColumns+` FROM account WHERE id = $1 FOR UPDATE`, id)
		if err != nil {
			return err
		}
		locked, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Account])
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		account = locked

		action, err := transition(account)
		if err != nil || action == "" {
			return err
		}
		if state == "CLOSED" {
			var pending bool
			err := tx.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM transfer
					WHERE (from_account_id = $1 OR to_account_id = $1)
					  AND state NOT IN ('COMPLETED', 'REJECTED', 'CANCELLED')
				)
			`, id).Scan(&pending)
			if err != nil {
				return err
			}
			if pending {
				return models.ErrAccountStateConflict
			}
		}

		if _, err := tx.Exec(ctx, `UPDATE account SET state = $2 WHERE id = $1`, id, state); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (actor_type, actor_id, action, account_id, details)
			VALUES ('ADMIN', $1, $2, $3, jsonb_build_object('previousState', $4::text, 'state', $5::text))
		`, actorID, action, id, account.State, state); err != nil {
			return err
		}
		account.State = state
		return nil
	})
	if err != nil {
		return models.Account{}, err
	}
	return account, nil
}

type AccountTx struct {
	tx      pgx.Tx
	ownerID string
}

func (r *AccountRepository) WithOwnerLock(ctx context.Context, ownerID string, run func(*AccountTx) error) error {
	return pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended('money.open_account:' || $1::text, 0))
		`, ownerID); err != nil {
			return err
		}

		return run(&AccountTx{tx: tx, ownerID: ownerID})
	})
}

func (a *AccountTx) Count(ctx context.Context) (int64, error) {
	var count int64
	err := a.tx.QueryRow(ctx, `
		SELECT count(*) FROM account WHERE owner_id = $1 AND type = 'CLIENT'
	`, a.ownerID).Scan(&count)
	return count, err
}

func (a *AccountTx) Create(ctx context.Context) (models.Account, error) {
	var account models.Account
	if err := a.tx.QueryRow(ctx, `
		INSERT INTO account (owner_id, type) VALUES ($1, 'CLIENT')
		RETURNING `+accountColumns+`
	`, a.ownerID).Scan(
		&account.ID, &account.OwnerID, &account.Type, &account.State,
		&account.Balance, &account.ReservedSum, &account.AvailableBalance, &account.OpenedAt,
	); err != nil {
		return models.Account{}, err
	}

	if _, err := a.tx.Exec(ctx, `
		INSERT INTO audit_log (actor_type, actor_id, action, account_id)
		VALUES ('USER', $1, 'ACCOUNT_OPENED', $2)
	`, a.ownerID, account.ID); err != nil {
		return models.Account{}, err
	}

	return account, nil
}
