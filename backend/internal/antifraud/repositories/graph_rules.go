package repositories

import (
	"context"
	"fmt"

	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type TransferEdge struct {
	SenderID    string
	RecipientID string
	Amount      decimal.Decimal
}

func CountDeviceUsers(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, windowSeconds int64) (int64, error) {
	var count int64
	err := tx.QueryRow(ctx, `
		SELECT count(DISTINCT sender.owner_id)
		FROM transfer_attempt AS t
		JOIN bank_account AS sender ON sender.id = t.from_account_id
		WHERE t.device = $1::text
		  AND t.created_at >= $2::timestamptz - $4::bigint * interval '1 second'
		  AND (t.created_at, t.id) <= ($2, $3)
	`, attempt.DeviceID, attempt.CreatedAt, attempt.TransferID, windowSeconds).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count device users: %w", err)
	}
	return count, nil
}

func ReadCycleEdges(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, windowSeconds int64) ([]TransferEdge, error) {
	// Only owners up to two steps from the current recipient can originate a return edge.
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE reachable(owner_id, depth) AS (
			VALUES ($1::text, 0)
			UNION
			SELECT recipient.owner_id, origin.depth + 1
			FROM reachable AS origin
			JOIN bank_account AS sender ON sender.owner_id = origin.owner_id
			JOIN transfer_attempt AS t ON t.from_account_id = sender.id
			JOIN bank_account AS recipient ON recipient.id = t.to_account_id
			WHERE origin.depth < 2
			  AND t.created_at >= $2::timestamptz - $5::bigint * interval '1 second'
			  AND (t.created_at, t.id) < ($2, $3)
			  AND recipient.owner_id NOT IN (origin.owner_id, $1, $4)
		)
		SELECT sender.owner_id AS sender_id, recipient.owner_id AS recipient_id, t.amount
		FROM (SELECT DISTINCT owner_id FROM reachable) AS origin
		JOIN bank_account AS sender ON sender.owner_id = origin.owner_id
		JOIN transfer_attempt AS t ON t.from_account_id = sender.id
		JOIN bank_account AS recipient ON recipient.id = t.to_account_id
		WHERE t.created_at >= $2::timestamptz - $5::bigint * interval '1 second'
		  AND (t.created_at, t.id) < ($2, $3)
		  AND recipient.owner_id NOT IN (sender.owner_id, $1)
	`, attempt.RecipientID, attempt.CreatedAt, attempt.TransferID, attempt.SenderID, windowSeconds)
	if err != nil {
		return nil, fmt.Errorf("query bounded transfer graph: %w", err)
	}
	edges, err := pgx.CollectRows(rows, pgx.RowToStructByName[TransferEdge])
	if err != nil {
		return nil, fmt.Errorf("read bounded transfer graph: %w", err)
	}
	return edges, nil
}
