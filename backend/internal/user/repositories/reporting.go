package repositories

import (
	"context"
	"time"

	"e-bank/backend/internal/user/models"

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
	err := r.pool.QueryRow(ctx, `
		SELECT count(*), count(DISTINCT keycloak_sub),
		       (SELECT count(*) FROM audit_log WHERE occurred_at >= $1 AND occurred_at < $2)
		FROM user_login
		WHERE logined_at >= $1 AND logined_at < $2
	`, reportingTimeBound(&from), reportingTimeBound(&to)).Scan(
		&result.LoginCount, &result.UniqueLoggedInUsers, &result.AdministrativeActionCount,
	)
	return result, err
}

func (r *ReportingRepository) Audit(ctx context.Context, filter models.AuditFilter) ([]models.AuditEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, occurred_at, actor_id, action, user_id, details
		FROM audit_log
		WHERE ($1::timestamptz IS NULL OR occurred_at >= $1)
		  AND ($2::timestamptz IS NULL OR occurred_at < $2)
		  AND ($3::text IS NULL OR actor_id = $3)
		  AND ($4::text IS NULL OR user_id = $4)
		  AND ($5::text IS NULL OR action = $5)
		ORDER BY occurred_at DESC, id DESC
		LIMIT $6 OFFSET $7
	`, reportingTimeBound(filter.From), reportingTimeBound(filter.To), filter.ActorID,
		filter.UserID, filter.Action, filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[models.AuditEvent])
}

// PostgreSQL timestamps have microsecond precision. Rounding bounds up preserves
// the half-open interval when the caller supplies a finer timestamp.
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
