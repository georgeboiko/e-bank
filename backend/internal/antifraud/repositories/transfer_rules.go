package repositories

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/antifraud/models"
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

func ReadTransferRuleFacts(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, configs models.RuleConfigs) (TransferRuleFacts, error) {
	var facts TransferRuleFacts
	err := tx.QueryRow(ctx, `
		WITH recent_attempts AS (
			SELECT t.to_account_id, t.created_at
			FROM bank_account AS sender
			JOIN transfer_attempt AS t ON t.from_account_id = sender.id
			WHERE sender.owner_id = $1
			  AND t.created_at >= $2::timestamptz - greatest($7::bigint, $8::bigint) * interval '1 second'
			  AND (t.created_at, t.id) <= ($2, $3)
		),
		recent_recipients AS (
			SELECT DISTINCT recipient.owner_id
			FROM recent_attempts AS t
			JOIN bank_account AS recipient ON recipient.id = t.to_account_id
			WHERE recipient.owner_id <> $1
			  AND t.created_at >= $2::timestamptz - $8::bigint * interval '1 second'
		)
		SELECT
			count(*) FILTER (WHERE created_at >= $2::timestamptz - $6::bigint * interval '1 second'),
			count(*) FILTER (WHERE created_at >= $2::timestamptz - $7::bigint * interval '1 second'),
			(
				SELECT count(*)
				FROM recent_recipients AS recent
				WHERE NOT EXISTS (
					SELECT 1
					FROM bank_account AS sender
					JOIN transfer_attempt AS t ON t.from_account_id = sender.id
					JOIN bank_account AS recipient ON recipient.id = t.to_account_id
					WHERE sender.owner_id = $1 AND recipient.owner_id = recent.owner_id
					  AND t.created_at < $2::timestamptz - $8::bigint * interval '1 second'
				)
			),
			(
				SELECT count(DISTINCT sender.owner_id)
				FROM transfer_attempt AS t
				JOIN bank_account AS sender ON sender.id = t.from_account_id
				WHERE t.to_account_id = $4
				  AND t.created_at >= $2::timestamptz - $9::bigint * interval '1 second'
				  AND (t.created_at, t.id) <= ($2, $3)
			),
			(SELECT min(first_seen_at) FROM user_device WHERE user_id = $1 AND device = $5::text),
			(SELECT min(first_seen_at) FROM user_device WHERE user_id = $1 AND device <> $5::text)
		FROM recent_attempts
	`, attempt.SenderID, attempt.CreatedAt, attempt.TransferID, attempt.ToAccountID, attempt.DeviceID,
		configs[0].Integer("shortWindowSeconds"), configs[0].Integer("longWindowSeconds"),
		configs[3].Integer("windowSeconds"), configs[4].Integer("windowSeconds")).Scan(
		&facts.ShortAttempts, &facts.LongAttempts, &facts.NewRecipients, &facts.SendersToAccount,
		&facts.FirstDeviceUse, &facts.FirstOtherDeviceUse,
	)
	if err != nil {
		return TransferRuleFacts{}, fmt.Errorf("read transfer rule facts: %w", err)
	}
	return facts, nil
}

func ReadApprovedTransferAmounts(ctx context.Context, tx pgx.Tx, attempt kafka.FraudCheckRequestedData, historyDays int64) ([]decimal.Decimal, error) {
	since := attempt.CreatedAt.UTC().AddDate(0, 0, -int(historyDays))
	// The event contract represents years with four digits.
	earliest := time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)
	if since.Before(earliest) {
		since = earliest
	}
	rows, err := tx.Query(ctx, `
		SELECT t.amount
		FROM bank_account AS sender
		JOIN transfer_attempt AS t ON t.from_account_id = sender.id
		JOIN antifraud_check AS check_result ON check_result.transfer_id = t.id
		WHERE sender.owner_id = $1 AND check_result.automatic_decision = 'APPROVED'
		  AND t.created_at >= $4::timestamptz
		  AND (t.created_at, t.id) < ($2, $3)
	`, attempt.SenderID, attempt.CreatedAt, attempt.TransferID, since)
	if err != nil {
		return nil, fmt.Errorf("query previously approved amounts: %w", err)
	}
	amounts, err := pgx.CollectRows(rows, pgx.RowTo[decimal.Decimal])
	if err != nil {
		return nil, fmt.Errorf("read previously approved amounts: %w", err)
	}
	return amounts, nil
}
