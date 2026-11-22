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

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.FraudDecisionData], error) {
	return postgres.ReadOutboxEvent[kafka.FraudDecisionData](ctx, pool, eventID, kafka.EventFraudDecision)
}

func PublishNextOutboxEvent(ctx context.Context, pool *pgxpool.Pool, publish func(transferID int32, payload []byte) error) (bool, error) {
	var attempted bool
	var publishErr error
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var eventID pgtype.UUID
		var transferID int32
		var payload []byte
		err := tx.QueryRow(ctx, `
			SELECT o.event_id, c.transfer_id, o.payload
			FROM outbox_event o
			JOIN antifraud_check c ON c.id = o.check_id
			WHERE o.published = false AND o.next_attempt_at <= now()
			  AND o.payload->>'eventType' = $1
			  AND o.payload#>>'{data,decision}' IN ($2, $3)
			  AND o.payload#>>'{data,source}' IN ($4, $5)
			ORDER BY o.next_attempt_at, o.created_at
			LIMIT 1
			FOR UPDATE OF o SKIP LOCKED
		`, kafka.EventFraudDecision, kafka.DecisionApproved, kafka.DecisionRejected,
			kafka.SourceAutomatic, kafka.SourceAnalyst).Scan(&eventID, &transferID, &payload)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select pending Antifraud outbox event: %w", err)
		}

		attempted = true
		if err := publish(transferID, payload); err != nil {
			publishErr = fmt.Errorf("publish Antifraud outbox event %s: %w", eventID.String(), err)
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
