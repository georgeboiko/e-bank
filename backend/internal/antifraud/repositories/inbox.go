package repositories

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StoreTransferAttempt(ctx context.Context, pool *pgxpool.Pool, event kafka.FraudCheckRequested, payload []byte) error {
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		apply, err := beginInboxEvent(ctx, tx, event.EventID, event.OccurredAt, payload)
		if err != nil || !apply {
			return err
		}

		data := event.Data
		if err := storeAccountOwners(ctx, tx, data); err != nil {
			return err
		}
		if err := storeAttempt(ctx, tx, data); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE inbox_event
			SET transfer_id = $2, applied = true, processed_at = clock_timestamp()
			WHERE event_id = $1::text::uuid
		`, event.EventID, data.TransferID)
		if err != nil {
			return fmt.Errorf("mark transfer request applied: %w", err)
		}
		return nil
	})
}

func beginInboxEvent(ctx context.Context, tx pgx.Tx, eventID string, occurredAt time.Time, payload []byte) (bool, error) {
	inbox, err := tx.Exec(ctx, `
		INSERT INTO inbox_event (event_id, occurred_at, payload)
		VALUES ($1::text::uuid, $2, $3::jsonb)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, occurredAt, payload)
	if err != nil {
		return false, fmt.Errorf("insert inbox event: %w", err)
	}
	if inbox.RowsAffected() != 0 {
		return true, nil
	}

	var matches, applied bool
	err = tx.QueryRow(ctx, `
		SELECT payload = $2::jsonb, applied
		FROM inbox_event WHERE event_id = $1::text::uuid FOR UPDATE
	`, eventID, payload).Scan(&matches, &applied)
	if err != nil {
		return false, fmt.Errorf("read existing inbox event: %w", err)
	}
	if !matches {
		return false, fmt.Errorf("inbox event %s was already received with a different payload", eventID)
	}
	return !applied, nil
}

func storeAccountOwners(ctx context.Context, tx pgx.Tx, data kafka.FraudCheckRequestedData) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO app_user (id)
		SELECT DISTINCT owner_id
		FROM (VALUES ($1::text), ($2::text)) AS owners(owner_id)
		ORDER BY owner_id
		ON CONFLICT (id) DO NOTHING
	`, data.SenderID, data.RecipientID)
	if err != nil {
		return fmt.Errorf("store transfer account owners: %w", err)
	}

	args := []any{data.FromAccountID, data.SenderID, data.ToAccountID, data.RecipientID}
	_, err = tx.Exec(ctx, `
		INSERT INTO bank_account (id, owner_id)
		SELECT id, owner_id
		FROM (VALUES ($1::integer, $2::text), ($3::integer, $4::text)) AS accounts(id, owner_id)
		ORDER BY id
		ON CONFLICT (id) DO NOTHING
	`, args...)
	if err != nil {
		return fmt.Errorf("store transfer accounts: %w", err)
	}

	var matches int
	err = tx.QueryRow(ctx, `
		SELECT count(*) FROM bank_account
		WHERE (id, owner_id) IN (($1::integer, $2::text), ($3::integer, $4::text))
	`, args...).Scan(&matches)
	if err != nil {
		return fmt.Errorf("check transfer account owners: %w", err)
	}
	if matches != 2 {
		return fmt.Errorf("transfer %d conflicts with the stored account ownership", data.TransferID)
	}
	return nil
}

func storeAttempt(ctx context.Context, tx pgx.Tx, data kafka.FraudCheckRequestedData) error {
	args := []any{data.TransferID, data.FromAccountID, data.ToAccountID, data.Amount.String(), data.DeviceID, data.IPAddress, data.CreatedAt}
	attempt, err := tx.Exec(ctx, `
		INSERT INTO transfer_attempt (id, from_account_id, to_account_id, amount, device, ip_address, created_at)
		VALUES ($1, $2, $3, $4::text::numeric, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, args...)
	if err != nil {
		return fmt.Errorf("insert transfer attempt: %w", err)
	}
	if attempt.RowsAffected() == 0 {
		var matches bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM transfer_attempt
				WHERE id = $1 AND from_account_id = $2 AND to_account_id = $3
				  AND amount = $4::text::numeric AND device IS NOT DISTINCT FROM $5
				  AND ip_address IS NOT DISTINCT FROM $6 AND created_at = $7
			)
		`, args...).Scan(&matches)
		if err != nil {
			return fmt.Errorf("check existing transfer attempt: %w", err)
		}
		if !matches {
			return fmt.Errorf("transfer attempt %d already exists with different data", data.TransferID)
		}
		return nil
	}

	if data.DeviceID != nil {
		_, err := tx.Exec(ctx, `
			INSERT INTO user_device (user_id, device, first_seen_at, last_seen_at, use_count)
			VALUES ($1, $2, $3, $3, 1)
			ON CONFLICT (user_id, device) DO UPDATE SET
				first_seen_at = least(user_device.first_seen_at, EXCLUDED.first_seen_at),
				last_seen_at = greatest(user_device.last_seen_at, EXCLUDED.last_seen_at),
				use_count = user_device.use_count + 1
		`, data.SenderID, *data.DeviceID, data.CreatedAt)
		if err != nil {
			return fmt.Errorf("record sender device usage: %w", err)
		}
	}
	return nil
}
