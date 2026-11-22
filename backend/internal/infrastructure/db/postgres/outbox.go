package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxEvent[T any] struct {
	Event   kafka.Event[T]
	Payload json.RawMessage
}

func ReadOutboxEvent[T any](ctx context.Context, pool *pgxpool.Pool, eventID string, eventType kafka.EventType) (OutboxEvent[T], error) {
	var payload json.RawMessage
	err := pool.QueryRow(ctx, `SELECT payload FROM outbox_event WHERE event_id = $1::text::uuid`, eventID).Scan(&payload)
	if err != nil {
		return OutboxEvent[T]{}, fmt.Errorf("read %s outbox event: %w", eventType, err)
	}
	event, err := kafka.Decode[T](payload, eventType)
	if err != nil {
		return OutboxEvent[T]{}, err
	}
	return OutboxEvent[T]{Event: event, Payload: payload}, nil
}
