package repositories

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/antifraud/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadAnalytics(ctx context.Context, pool *pgxpool.Pool, from, to time.Time, topLimit int) (models.Analytics, error) {
	result := models.Analytics{From: from, To: to}
	err := pool.QueryRow(ctx, `
		WITH checks AS (
			SELECT id, transfer_id, risk_score, automatic_decision, checked_at
			FROM antifraud_check WHERE checked_at >= $1 AND checked_at < $2
		), attempts AS (
			SELECT t.device, t.from_account_id, t.to_account_id, t.amount,
			       sender.owner_id AS sender_id, recipient.owner_id AS recipient_id
			FROM transfer_attempt t
			JOIN bank_account sender ON sender.id = t.from_account_id
			JOIN bank_account recipient ON recipient.id = t.to_account_id
			WHERE t.created_at >= $1 AND t.created_at < $2
		), totals AS (
			SELECT count(*) FILTER (WHERE c.automatic_decision = 'APPROVED') AS approved,
			       count(*) FILTER (WHERE c.automatic_decision = 'REVIEW_REQUIRED') AS review,
			       count(*) FILTER (WHERE c.automatic_decision = 'REJECTED') AS rejected,
			       avg(extract(epoch FROM (c.checked_at - t.received_at)) * 1000)::double precision AS average_ms
			FROM checks c JOIN transfer_attempt t ON t.id = c.transfer_id
		), trigger_counts AS (
			SELECT r.rule_id, count(*) AS count FROM checks c
			JOIN rule_evaluation r ON r.check_id = c.id
			WHERE r.status = 'TRIGGERED' GROUP BY r.rule_id
		), risk_users AS (
			SELECT sender.owner_id AS user_id, count(*) AS warning_count, max(c.risk_score) AS max_score
			FROM checks c
			JOIN transfer_attempt t ON t.id = c.transfer_id
			JOIN bank_account sender ON sender.id = t.from_account_id
			WHERE c.risk_score >= 30
			GROUP BY sender.owner_id
			ORDER BY warning_count DESC, user_id
			LIMIT $3
		), devices AS (
			SELECT device, count(DISTINCT sender_id) AS user_count, count(*) AS attempt_count
			FROM attempts WHERE device IS NOT NULL
			GROUP BY device HAVING count(DISTINCT sender_id) >= 2
			ORDER BY user_count DESC, device
			LIMIT $3
		), account_flows AS (
			SELECT to_account_id AS account_id, sender_id AS counterparty, amount, true AS incoming FROM attempts
			UNION ALL
			SELECT from_account_id, recipient_id, amount, false FROM attempts
		), account_counts AS (
			SELECT account_id,
			       count(DISTINCT counterparty) FILTER (WHERE incoming) AS incoming_clients,
			       count(DISTINCT counterparty) FILTER (WHERE NOT incoming) AS outgoing_clients,
			       coalesce(sum(amount) FILTER (WHERE incoming), 0.00)::text AS incoming_amount,
			       coalesce(sum(amount) FILTER (WHERE NOT incoming), 0.00)::text AS outgoing_amount
			FROM account_flows GROUP BY account_id
			ORDER BY count(DISTINCT counterparty) FILTER (WHERE incoming)
			         + count(DISTINCT counterparty) FILTER (WHERE NOT incoming) DESC, account_id
			LIMIT $3
		)
		SELECT approved, review, rejected,
		       (SELECT count(*) FROM review_case WHERE status IN ('OPEN', 'IN_PROGRESS')),
		       average_ms,
		       ARRAY(SELECT coalesce(t.count, 0) FROM generate_series(1, 8) AS ids(id)
		             LEFT JOIN trigger_counts t ON t.rule_id = ids.id ORDER BY ids.id),
		       coalesce((SELECT jsonb_agg(jsonb_build_object('userId', user_id, 'warningCount', warning_count, 'maxScore', max_score)
		                                  ORDER BY warning_count DESC, user_id) FROM risk_users), '[]'::jsonb),
		       coalesce((SELECT jsonb_agg(jsonb_build_object('deviceId', device, 'userCount', user_count, 'attemptCount', attempt_count)
		                                  ORDER BY user_count DESC, device) FROM devices), '[]'::jsonb),
		       coalesce((SELECT jsonb_agg(jsonb_build_object('accountId', account_id, 'incomingClients', incoming_clients,
		                                  'outgoingClients', outgoing_clients, 'incomingAttemptedAmount', incoming_amount,
		                                  'outgoingAttemptedAmount', outgoing_amount)
		                                  ORDER BY incoming_clients + outgoing_clients DESC, account_id) FROM account_counts), '[]'::jsonb)
		FROM totals
	`, reportingTimeBound(&from), reportingTimeBound(&to), topLimit).Scan(
		&result.Approved, &result.Review, &result.Rejected, &result.OpenReviews, &result.AverageMilliseconds,
		&result.RuleTriggers, &result.RiskUsers, &result.SharedDevices, &result.AccountsByCounterpartyCount,
	)
	if err != nil {
		return models.Analytics{}, fmt.Errorf("read antifraud analytics: %w", err)
	}
	return result, nil
}

func CheckExists(ctx context.Context, pool *pgxpool.Pool, id int32) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM antifraud_check WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func ReadAudit(ctx context.Context, pool *pgxpool.Pool, filter models.AuditFilter) ([]models.AuditEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id::text, a.occurred_at, a.actor_type, a.actor_id, a.action,
		       CASE
		           WHEN a.action = 'RULE_UPDATED' THEN 'RULE'
		           WHEN a.action IN ('REVIEW_CLAIMED', 'REVIEW_CLOSED') THEN 'REVIEW_CASE'
		           WHEN a.check_id IS NOT NULL THEN 'ANTIFRAUD_CHECK'
		           ELSE 'ANTIFRAUD'
		       END,
		       CASE
		           WHEN a.action = 'RULE_UPDATED' THEN a.details->>'ruleId'
		           WHEN a.action IN ('REVIEW_CLAIMED', 'REVIEW_CLOSED') THEN coalesce(a.details->>'reviewCaseId', r.id::text)
		           ELSE a.check_id::text
		       END,
		       a.details
		FROM audit_log a
		LEFT JOIN review_case r ON r.check_id = a.check_id AND a.action IN ('REVIEW_CLAIMED', 'REVIEW_CLOSED')
		WHERE ($1::timestamptz IS NULL OR a.occurred_at >= $1)
		  AND ($2::timestamptz IS NULL OR a.occurred_at < $2)
		  AND ($3::text IS NULL OR a.actor_id = $3)
		  AND ($4::integer IS NULL OR a.check_id = $4)
		  AND ($5::text IS NULL OR a.action = $5)
		ORDER BY a.occurred_at DESC, a.id DESC
		LIMIT $6 OFFSET $7
	`, reportingTimeBound(filter.From), reportingTimeBound(filter.To), filter.ActorID, filter.CheckID, filter.Action, filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("read antifraud audit: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[models.AuditEvent])
}

// PostgreSQL timestamps are stored at microsecond precision. Rounding the bounds
// up preserves the half-open interval when an input has sub-microsecond precision.
func reportingTimeBound(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	bound := value.Truncate(time.Microsecond)
	if bound.Before(*value) {
		bound = bound.Add(time.Microsecond)
	}
	return &bound
}
