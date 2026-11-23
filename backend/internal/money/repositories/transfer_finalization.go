package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LastPendingFinalizationID(ctx context.Context, pool *pgxpool.Pool) (int32, error) {
	var id int32
	err := pool.QueryRow(ctx, `
		SELECT coalesce(max(id), 0)
		FROM transfer WHERE state IN ('COMPLETING', 'RELEASING_FUNDS')
	`).Scan(&id)
	return id, err
}

func FinalizeNextTransfer(ctx context.Context, pool *pgxpool.Pool, afterID, lastID int32) (int32, error) {
	var transferID int32
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var state string
		err := tx.QueryRow(ctx, `
			SELECT id, state
			FROM transfer
			WHERE state IN ('COMPLETING', 'RELEASING_FUNDS')
			  AND id > $1 AND id <= $2
			ORDER BY id
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, afterID, lastID).Scan(&transferID, &state)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select transfer to finalize: %w", err)
		}

		switch state {
		case "COMPLETING":
			_, err = tx.Exec(ctx, `SELECT commit_transfer($1)`, transferID)
		case "RELEASING_FUNDS":
			_, err = tx.Exec(ctx, `SELECT release_funds($1)`, transferID)
		default:
			return fmt.Errorf("unexpected transfer state %q", state)
		}
		return err
	})
	return transferID, err
}
