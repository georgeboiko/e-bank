package repositories

import (
	"context"
	"fmt"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SaveCheck(ctx context.Context, tx pgx.Tx, transferID int32, rules []models.RuleEvaluation) error {
	var scores [8]int32
	var statuses, reasons [8]string
	for _, rule := range rules {
		if rule.RuleID < 1 || rule.RuleID > 8 || statuses[rule.RuleID-1] != "" {
			return fmt.Errorf("invalid or repeated Antifraud rule %d", rule.RuleID)
		}
		i := rule.RuleID - 1
		scores[i], statuses[i], reasons[i] = rule.Score, rule.Status, rule.Reason
	}
	for i, status := range statuses {
		if status == "" {
			return fmt.Errorf("missing Antifraud rule %d", i+1)
		}
	}

	_, err := tx.Exec(ctx, `SELECT save_antifraud_check($1, $2, $3, $4, uuidv7())`,
		transferID, scores[:], statuses[:], reasons[:])
	if err != nil {
		return fmt.Errorf("save Antifraud check: %w", err)
	}
	return nil
}

type UncheckedTransferRequest struct {
	TransferID int32
	Payload    []byte
}

func ListUncheckedTransferRequests(ctx context.Context, pool *pgxpool.Pool) ([]UncheckedTransferRequest, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, i.payload
		FROM inbox_event i
		JOIN transfer_attempt t ON t.id = i.transfer_id
		WHERE i.applied AND i.payload->>'eventType' = $1
		  AND NOT EXISTS (SELECT 1 FROM antifraud_check c WHERE c.transfer_id = t.id)
		ORDER BY t.created_at, t.id
		LIMIT 100
	`, kafka.EventFraudCheckRequested)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[UncheckedTransferRequest])
}
