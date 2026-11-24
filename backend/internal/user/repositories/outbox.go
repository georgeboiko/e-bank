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

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.LoginSucceededData], error) {
	return postgres.ReadOutboxEvent[kafka.LoginSucceededData](ctx, pool, eventID, kafka.EventLoginSucceeded)
}

func PublishNextOutboxEvent(ctx context.Context, pool *pgxpool.Pool, publish func(userID string, payload []byte) error) (bool, error) {
	var attempted bool
	var publishErr error
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var eventID pgtype.UUID
		var userID string
		var payload []byte
		err := tx.QueryRow(ctx, `
			SELECT event.event_id, login.keycloak_sub, event.payload
			FROM outbox_event AS event
			JOIN user_login AS login ON login.id = event.user_login_id
			WHERE event.published = false AND event.next_attempt_at <= now()
			  AND event.payload->>'eventType' = $1
			ORDER BY event.next_attempt_at, event.created_at
			LIMIT 1
			FOR UPDATE OF event SKIP LOCKED
		`, kafka.EventLoginSucceeded).Scan(&eventID, &userID, &payload)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select pending User outbox event: %w", err)
		}

		attempted = true
		if err := publish(userID, payload); err != nil {
			publishErr = fmt.Errorf("publish User outbox event %s: %w", eventID.String(), err)
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
