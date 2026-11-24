package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminAuditRepository struct {
	pool *pgxpool.Pool
}

func NewAdminAuditRepository(pool *pgxpool.Pool) *AdminAuditRepository {
	return &AdminAuditRepository{pool: pool}
}

type AdminAuditTx struct {
	tx pgx.Tx
}

func (r *AdminAuditRepository) WithUserLock(ctx context.Context, userID string, work func(*AdminAuditTx) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended('user.admin:' || $1::text, 0))
		`, userID); err != nil {
			return err
		}
		return work(&AdminAuditTx{tx: tx})
	})
}

func (r *AdminAuditTx) Record(ctx context.Context, actorID, userID, action string, details map[string]any) error {
	_, err := r.tx.Exec(ctx, `
		INSERT INTO audit_log (occurred_at, actor_id, user_id, action, details)
		VALUES (clock_timestamp(), $1, $2, $3, $4)
	`, actorID, userID, action, details)
	return err
}
