package repositories

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type TransferRuleFacts struct {
	ShortAttempts       int64
	LongAttempts        int64
	NewRecipients       int64
	SendersToAccount    int64
	FirstDeviceUse      *time.Time
	FirstOtherDeviceUse *time.Time
}

func ReadTransferRuleFacts(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData) (TransferRuleFacts, error) {
	var facts TransferRuleFacts
	err := tx.QueryRow(ctx, `
		WITH recent_attempts AS (
			SELECT t.to_account_id, t.created_at
			FROM bank_account AS sender
			JOIN transfer_attempt AS t ON t.from_account_id = sender.id
			WHERE sender.owner_id = $1
			  AND t.created_at >= $2::timestamptz - interval '10 minutes'
			  AND (t.created_at, t.id) <= ($2, $3)
		),
		recent_recipients AS (
			SELECT DISTINCT recipient.owner_id
			FROM recent_attempts AS t
			JOIN bank_account AS recipient ON recipient.id = t.to_account_id
			WHERE recipient.owner_id <> $1
		)
		SELECT
			count(*) FILTER (WHERE created_at >= $2::timestamptz - interval '2 minutes'),
			count(*),
			(
				SELECT count(*)
				FROM recent_recipients AS recent
				WHERE NOT EXISTS (
					SELECT 1
					FROM bank_account AS sender
					JOIN transfer_attempt AS t ON t.from_account_id = sender.id
					JOIN bank_account AS recipient ON recipient.id = t.to_account_id
					WHERE sender.owner_id = $1 AND recipient.owner_id = recent.owner_id
					  AND t.created_at < $2::timestamptz - interval '10 minutes'
				)
			),
			(
				SELECT count(DISTINCT sender.owner_id)
				FROM transfer_attempt AS t
				JOIN bank_account AS sender ON sender.id = t.from_account_id
				WHERE t.to_account_id = $4
				  AND t.created_at >= $2::timestamptz - interval '1 hour'
				  AND (t.created_at, t.id) <= ($2, $3)
			),
			(SELECT min(first_seen_at) FROM user_device WHERE user_id = $1 AND device = $5::text),
			(SELECT min(first_seen_at) FROM user_device WHERE user_id = $1 AND device <> $5::text)
		FROM recent_attempts
	`, attempt.SenderID, attempt.CreatedAt, attempt.TransferID, attempt.ToAccountID, attempt.DeviceID).Scan(
		&facts.ShortAttempts, &facts.LongAttempts, &facts.NewRecipients, &facts.SendersToAccount,
		&facts.FirstDeviceUse, &facts.FirstOtherDeviceUse,
	)
	if err != nil {
		return TransferRuleFacts{}, fmt.Errorf("read transfer rule facts: %w", err)
	}
	return facts, nil
}

func ReadApprovedTransferAmounts(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData) ([]decimal.Decimal, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.amount
		FROM bank_account AS sender
		JOIN transfer_attempt AS t ON t.from_account_id = sender.id
		JOIN antifraud_check AS check_result ON check_result.transfer_id = t.id
		WHERE sender.owner_id = $1 AND check_result.automatic_decision = 'APPROVED'
		  AND t.created_at >= $2::timestamptz - interval '720 hours'
		  AND (t.created_at, t.id) < ($2, $3)
	`, attempt.SenderID, attempt.CreatedAt, attempt.TransferID)
	if err != nil {
		return nil, fmt.Errorf("query previously approved amounts: %w", err)
	}
	amounts, err := pgx.CollectRows(rows, pgx.RowTo[decimal.Decimal])
	if err != nil {
		return nil, fmt.Errorf("read previously approved amounts: %w", err)
	}
	return amounts, nil
}
