package repositories

import (
	"context"
	"fmt"

	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StoreLogin(ctx context.Context, pool *pgxpool.Pool, event kafka.LoginSucceeded, payload []byte) error {
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		apply, err := beginInboxEvent(ctx, tx, event.EventID, event.OccurredAt, payload)
		if err != nil || !apply {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO app_user (id) VALUES ($1) ON CONFLICT (id) DO NOTHING
		`, event.Data.UserID); err != nil {
			return fmt.Errorf("store login user: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO login_event (user_id, ip_address, occurred_at)
			VALUES ($1, $2, $3)
		`, event.Data.UserID, event.Data.IPAddress, event.OccurredAt); err != nil {
			return fmt.Errorf("store login event: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inbox_event SET applied = true, processed_at = clock_timestamp()
			WHERE event_id = $1::text::uuid
		`, event.EventID); err != nil {
			return fmt.Errorf("mark login event applied: %w", err)
		}
		return nil
	})
}
