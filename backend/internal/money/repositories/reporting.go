package repositories

import (
	"context"
	"time"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportingRepository struct {
	pool *pgxpool.Pool
}

func NewReportingRepository(pool *pgxpool.Pool) *ReportingRepository {
	return &ReportingRepository{pool: pool}
}

func (r *ReportingRepository) Analytics(ctx context.Context, from, to time.Time) (models.Analytics, error) {
	result := models.Analytics{From: from, To: to}
	// Round up to PostgreSQL timestamp precision to preserve the half-open interval.
	start := from.Add(time.Microsecond - time.Nanosecond).Truncate(time.Microsecond)
	end := to.Add(time.Microsecond - time.Nanosecond).Truncate(time.Microsecond)

	err := r.pool.QueryRow(ctx, `
		WITH created AS (
			SELECT count(*) AS total,
			       count(*) FILTER (WHERE state = 'PENDING') AS pending,
			       count(*) FILTER (WHERE state = 'FRAUD_CHECK_PENDING') AS fraud_check_pending,
			       count(*) FILTER (WHERE state = 'COMPLETING') AS completing,
			       count(*) FILTER (WHERE state = 'RELEASING_FUNDS') AS releasing_funds,
			       count(*) FILTER (WHERE state = 'COMPLETED') AS completed,
			       count(*) FILTER (WHERE state = 'REJECTED') AS rejected,
			       count(*) FILTER (WHERE state = 'CANCELLED') AS cancelled
			FROM transfer
			WHERE created_at >= $1 AND created_at < $2
		), turnover AS (
			SELECT coalesce(sum(t.amount), 0) AS total,
			       coalesce(sum(t.amount) FILTER (WHERE sender.type = 'TECHNICAL'), 0) AS topups
			FROM transfer t
			JOIN account sender ON sender.id = t.from_account_id
			WHERE t.state = 'COMPLETED' AND t.finished_at >= $1 AND t.finished_at < $2
		), accounts AS (
			SELECT count(*) FILTER (WHERE state = 'ACTIVE') AS active,
			       count(*) FILTER (WHERE state = 'BLOCKED') AS blocked,
			       count(*) FILTER (WHERE state = 'CLOSED') AS closed
			FROM account WHERE type = 'CLIENT'
		)
		SELECT created.total, pending, fraud_check_pending, completing, releasing_funds,
		       completed, rejected, cancelled, turnover.total, topups, active, blocked, closed
		FROM created CROSS JOIN turnover CROSS JOIN accounts
	`, start, end).Scan(
		&result.CreatedTransferCount,
		&result.Transfers.Pending, &result.Transfers.FraudCheckPending, &result.Transfers.Completing,
		&result.Transfers.ReleasingFunds, &result.Transfers.Completed, &result.Transfers.Rejected,
		&result.Transfers.Cancelled, &result.Turnover, &result.TopupTurnover,
		&result.Accounts.Active, &result.Accounts.Blocked, &result.Accounts.Closed,
	)
	return result, err
}

func (r *ReportingRepository) Audit(ctx context.Context, filter models.AuditFilter) ([]models.AuditEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id::text, a.occurred_at, a.actor_type, a.actor_id, a.action,
		       CASE
		           WHEN a.action = 'LIMIT_CHANGED' THEN 'TRANSFER_LIMITS'
		           WHEN a.transfer_id IS NOT NULL THEN 'TRANSFER'
		           ELSE 'ACCOUNT'
		       END,
		       CASE
		           WHEN a.action = 'LIMIT_CHANGED' THEN '1'
		           ELSE coalesce(a.transfer_id::text, a.account_id::text)
		       END,
		       a.details
		FROM audit_log a
		LEFT JOIN transfer t ON t.id = a.transfer_id AND $4::integer IS NOT NULL
		WHERE ($1::timestamptz IS NULL OR a.occurred_at >= $1)
		  AND ($2::timestamptz IS NULL OR a.occurred_at < $2)
		  AND ($3::text IS NULL OR a.actor_id = $3)
		  AND ($4::integer IS NULL OR $4 IN (a.account_id, t.from_account_id, t.to_account_id))
		  AND ($5::text IS NULL OR a.action = $5)
		ORDER BY a.occurred_at DESC, a.id DESC
		LIMIT $6 OFFSET $7
	`, filter.From, filter.To, filter.ActorID, filter.AccountID, filter.Action, filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[models.AuditEvent])
}
