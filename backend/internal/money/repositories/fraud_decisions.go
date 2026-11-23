package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyFraudDecision(ctx context.Context, pool *pgxpool.Pool, payload []byte) error {
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CALL apply_antifraud_decision($1::jsonb)`, payload)
		return err
	}); err != nil {
		return fmt.Errorf("apply fraud decision: %w", err)
	}
	return nil
}
