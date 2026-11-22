package repositories

import (
	"context"
	"errors"
	"fmt"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.FraudCheckRequestedData], error) {
	return postgres.ReadOutboxEvent[kafka.FraudCheckRequestedData](ctx, pool, eventID, kafka.EventFraudCheckRequested)
}

func PublishNextOutboxEvent(ctx context.Context, pool *pgxpool.Pool, publish func(transferID int32, payload []byte) error) (bool, error) {
	var attempted bool
	var publishErr error
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var eventID pgtype.UUID
		var transferID int32
		var payload []byte
		err := tx.QueryRow(ctx, `
			SELECT event_id, transfer_id, payload
			FROM outbox_event
			WHERE published = false AND next_attempt_at <= now()
			  AND payload->>'eventType' = $1
			ORDER BY next_attempt_at, created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, kafka.EventFraudCheckRequested).Scan(&eventID, &transferID, &payload)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select pending Money outbox event: %w", err)
		}

		attempted = true
		if err := publish(transferID, payload); err != nil {
			publishErr = fmt.Errorf("publish Money outbox event %s: %w", eventID.String(), err)
			_, err = tx.Exec(ctx, `
				UPDATE outbox_event
				SET attempt_count = attempt_count + 1, last_error = $2,
				    next_attempt_at = clock_timestamp() + interval '5 seconds'
				WHERE event_id = $1
			`, eventID, err.Error())
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE outbox_event
			SET published = true, published_at = clock_timestamp(),
			    attempt_count = attempt_count + 1, last_error = NULL
			WHERE event_id = $1
		`, eventID)
		return err
	})
	return attempted, errors.Join(err, publishErr)
}
