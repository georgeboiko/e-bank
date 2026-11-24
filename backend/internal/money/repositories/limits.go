package repositories

import (
	"context"
	"errors"

	"e-bank/backend/internal/money/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransferLimitRepository struct {
	pool *pgxpool.Pool
}

func NewTransferLimitRepository(pool *pgxpool.Pool) *TransferLimitRepository {
	return &TransferLimitRepository{pool: pool}
}

func (r *TransferLimitRepository) Get(ctx context.Context) (models.TransferLimits, error) {
	var limits models.TransferLimits
	err := r.pool.QueryRow(ctx, `SELECT daily_limit, monthly_limit FROM transfer_limits WHERE id = 1`).
		Scan(&limits.DailyLimit, &limits.MonthlyLimit)
	return limits, missingTransferLimits(err)
}

func (r *TransferLimitRepository) Update(ctx context.Context, actorID string, limits models.TransferLimits) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var previous models.TransferLimits
		err := tx.QueryRow(ctx, `
			SELECT daily_limit, monthly_limit FROM transfer_limits WHERE id = 1 FOR UPDATE
		`).Scan(&previous.DailyLimit, &previous.MonthlyLimit)
		if err != nil {
			return missingTransferLimits(err)
		}
		if previous.DailyLimit.Equal(limits.DailyLimit) && previous.MonthlyLimit.Equal(limits.MonthlyLimit) {
			return nil
		}

		if _, err := tx.Exec(ctx, `
			UPDATE transfer_limits SET daily_limit = $1, monthly_limit = $2 WHERE id = 1
		`, limits.DailyLimit, limits.MonthlyLimit); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO audit_log (occurred_at, actor_type, actor_id, action, details)
			VALUES (clock_timestamp(), 'ADMIN', $1, 'LIMIT_CHANGED', jsonb_build_object(
				'before', jsonb_build_object('dailyLimit', $2::text, 'monthlyLimit', $3::text),
				'after', jsonb_build_object('dailyLimit', $4::text, 'monthlyLimit', $5::text)
			))
		`, actorID, previous.DailyLimit.StringFixed(2), previous.MonthlyLimit.StringFixed(2),
			limits.DailyLimit.StringFixed(2), limits.MonthlyLimit.StringFixed(2))
		return err
	})
}

func (r *TransferLimitRepository) GetUsage(ctx context.Context, ownerID string) (models.TransferLimitUsage, error) {
	var usage models.TransferLimitUsage
	err := r.pool.QueryRow(ctx, `
		SELECT l.daily_limit, u.daily_used, u.daily_resets_at,
		       l.monthly_limit, u.monthly_used, u.monthly_resets_at
		FROM transfer_limits AS l
		CROSS JOIN transfer_limit_usage($1, statement_timestamp()) AS u
		WHERE l.id = 1
	`, ownerID).Scan(&usage.Daily.Limit, &usage.Daily.Used, &usage.Daily.ResetsAt,
		&usage.Monthly.Limit, &usage.Monthly.Used, &usage.Monthly.ResetsAt)
	return usage, missingTransferLimits(err)
}

func missingTransferLimits(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrLimitsNotConfigured
	}
	return err
}
