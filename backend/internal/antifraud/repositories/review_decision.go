package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/antifraud/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewDecisionRepository struct {
	pool *pgxpool.Pool
}

func NewReviewDecisionRepository(pool *pgxpool.Pool) *ReviewDecisionRepository {
	return &ReviewDecisionRepository{pool: pool}
}

func (r *ReviewDecisionRepository) Claim(ctx context.Context, caseID int32, analystID string) (models.ReviewCase, error) {
	var review models.ReviewCase
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		var checkID int32
		var status string
		var assignedTo *string
		err := tx.QueryRow(ctx, `
			SELECT check_id, status, analyst_id
			FROM review_case WHERE id = $1 FOR UPDATE
		`, caseID).Scan(&checkID, &status, &assignedTo)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.ErrReviewCaseNotFound
		case err != nil:
			return err
		case status == "CLOSED":
			return models.ErrReviewCaseClosed
		case assignedTo != nil && *assignedTo != analystID:
			return models.ErrReviewCaseAssigned
		}

		if status == "OPEN" {
			if _, err := tx.Exec(ctx, `INSERT INTO app_user (id) VALUES ($1) ON CONFLICT DO NOTHING`, analystID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE review_case SET analyst_id = $2, status = 'IN_PROGRESS' WHERE id = $1
			`, caseID, analystID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO audit_log (occurred_at, actor_type, actor_id, action, check_id, details)
				VALUES (clock_timestamp(), 'ANALYST', $1, 'REVIEW_CLAIMED', $2,
				        jsonb_build_object('reviewCaseId', $3::integer))
			`, analystID, checkID, caseID); err != nil {
				return err
			}
		}
		review, err = readReviewCaseAfterMutation(ctx, tx, caseID)
		return err
	})
	if err != nil {
		return models.ReviewCase{}, err
	}
	return review, nil
}

func (r *ReviewDecisionRepository) Resolve(ctx context.Context, caseID int32, analystID, decision string, comment *string) (models.ReviewCase, error) {
	var review models.ReviewCase
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CALL resolve_review_case($1, $2, $3, $4, uuidv7())`, caseID, analystID, decision, comment)
		if err != nil {
			return resolveReviewCaseError(err)
		}
		review, err = readReviewCaseAfterMutation(ctx, tx, caseID)
		return err
	})
	if err != nil {
		return models.ReviewCase{}, err
	}
	return review, nil
}

func readReviewCaseAfterMutation(ctx context.Context, tx pgx.Tx, caseID int32) (models.ReviewCase, error) {
	rows, err := tx.Query(ctx, `
		SELECT r.id, r.check_id, c.transfer_id, r.analyst_id, r.status, r.decision, r.comment, r.created_at, r.decided_at
		FROM review_case AS r
		JOIN antifraud_check AS c ON c.id = r.check_id
		WHERE r.id = $1
	`, caseID)
	if err != nil {
		return models.ReviewCase{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.ReviewCase])
}

func resolveReviewCaseError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "P0002":
		return models.ErrReviewCaseNotFound
	case "P3321":
		return models.ErrReviewCaseAssigned
	case "P3322":
		return models.ErrReviewCaseClosed
	default:
		return err
	}
}
