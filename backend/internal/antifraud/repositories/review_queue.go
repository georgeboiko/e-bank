package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/antifraud/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const checkSummaryColumns = `
	c.id, c.transfer_id, c.risk_score, c.automatic_decision,
	CASE
		WHEN c.automatic_decision <> 'REVIEW_REQUIRED' THEN c.automatic_decision
		WHEN r.status = 'CLOSED' THEN r.decision
	END AS final_decision,
	CASE
		WHEN c.automatic_decision <> 'REVIEW_REQUIRED' THEN 'AUTOMATIC'
		WHEN r.status = 'CLOSED' THEN 'ANALYST'
	END AS decision_source,
	r.id AS review_case_id, c.checked_at`

const checkSummaryJoins = `
	FROM antifraud_check c
	JOIN transfer_attempt t ON t.id = c.transfer_id
	JOIN bank_account sender ON sender.id = t.from_account_id
	JOIN bank_account recipient ON recipient.id = t.to_account_id
	LEFT JOIN review_case r ON r.check_id = c.id`

const reviewCaseColumns = `
	r.id, r.check_id, c.transfer_id, r.analyst_id, r.status,
	r.decision, r.comment, r.created_at, r.decided_at`

type ReviewQueueRepository struct {
	pool *pgxpool.Pool
}

func NewReviewQueueRepository(pool *pgxpool.Pool) *ReviewQueueRepository {
	return &ReviewQueueRepository{pool: pool}
}

func (r *ReviewQueueRepository) ListChecks(ctx context.Context, filter models.CheckListFilter) ([]models.CheckSummary, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+checkSummaryColumns+checkSummaryJoins+`
		WHERE ($1::integer IS NULL OR c.transfer_id = $1)
		  AND ($2::text IS NULL OR $2 IN (sender.owner_id, recipient.owner_id))
		  AND ($3::integer IS NULL OR t.from_account_id = $3)
		  AND ($4::integer IS NULL OR t.to_account_id = $4)
		  AND ($5::text = '' OR c.automatic_decision = $5)
		  AND ($6::integer IS NULL OR c.risk_score >= $6)
		  AND ($7::integer IS NULL OR c.risk_score <= $7)
		  AND ($8::timestamptz IS NULL OR c.checked_at >= $8)
		  AND ($9::timestamptz IS NULL OR c.checked_at < $9)
		ORDER BY c.checked_at DESC, c.id DESC
		LIMIT $10 OFFSET $11
	`, filter.TransferID, filter.UserID, filter.FromAccountID, filter.ToAccountID,
		filter.AutomaticDecision, filter.MinScore, filter.MaxScore, filter.From, filter.To,
		filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.CheckSummary])
}

func (r *ReviewQueueRepository) GetCheck(ctx context.Context, id int32) (models.Check, error) {
	var check models.Check
	t := &check.Transfer
	err := r.pool.QueryRow(ctx, `SELECT `+checkSummaryColumns+`,
		t.id, t.from_account_id, t.to_account_id, sender.owner_id, recipient.owner_id,
		t.amount::text, t.device, host(t.ip_address), t.created_at, t.received_at,
		t.geo_status, t.latitude::float8, t.longitude::float8, t.geo_resolved_at
	`+checkSummaryJoins+` WHERE c.id = $1`, id).Scan(
		&check.ID, &check.TransferID, &check.RiskScore, &check.AutomaticDecision,
		&check.FinalDecision, &check.DecisionSource, &check.ReviewCaseID, &check.CheckedAt,
		&t.ID, &t.FromAccountID, &t.ToAccountID, &t.SenderID, &t.RecipientID,
		&t.Amount, &t.DeviceID, &t.IPAddress, &t.CreatedAt, &t.ReceivedAt,
		&t.Geo.Status, &t.Geo.Latitude, &t.Geo.Longitude, &t.Geo.ResolvedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Check{}, models.ErrCheckNotFound
	}
	return check, err
}

func (r *ReviewQueueRepository) ListCheckRules(ctx context.Context, id int32) ([]models.RuleEvaluation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rule_id, status, score, reason
		FROM rule_evaluation
		WHERE check_id = $1
		ORDER BY rule_id
	`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.RuleEvaluation])
}

func (r *ReviewQueueRepository) ListReviewCases(ctx context.Context, filter models.ReviewCaseListFilter) ([]models.ReviewCase, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+reviewCaseColumns+`
		FROM review_case r
		JOIN antifraud_check c ON c.id = r.check_id
		WHERE (CASE WHEN $1::text = '' THEN r.status IN ('OPEN', 'IN_PROGRESS') ELSE r.status = $1 END)
		  AND ($2::text IS NULL OR r.analyst_id = $2)
		  AND ($3::timestamptz IS NULL OR r.created_at >= $3)
		  AND ($4::timestamptz IS NULL OR r.created_at < $4)
		ORDER BY r.created_at ASC, r.id ASC
		LIMIT $5 OFFSET $6
	`, filter.Status, filter.AnalystID, filter.From, filter.To, filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.ReviewCase])
}

func (r *ReviewQueueRepository) GetReviewCase(ctx context.Context, id int32) (models.ReviewCase, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+reviewCaseColumns+`
		FROM review_case r
		JOIN antifraud_check c ON c.id = r.check_id
		WHERE r.id = $1
	`, id)
	if err != nil {
		return models.ReviewCase{}, err
	}
	review, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.ReviewCase])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ReviewCase{}, models.ErrReviewCaseNotFound
	}
	return review, err
}
