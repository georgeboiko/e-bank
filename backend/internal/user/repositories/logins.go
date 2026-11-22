package repositories

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LatestLoginAt(ctx context.Context, pool *pgxpool.Pool) (time.Time, error) {
	var latest *time.Time
	err := pool.QueryRow(ctx, "SELECT max(logined_at) FROM user_login").Scan(&latest)
	if err != nil || latest == nil {
		return time.Time{}, err
	}
	return *latest, nil
}

func RecordLogin(ctx context.Context, pool *pgxpool.Pool, eventID [16]byte, userID string, ip netip.Addr, at time.Time) error {
	_, err := pool.Exec(ctx, `
		SELECT record_login($1, $2, $3, $4)
		WHERE NOT EXISTS (SELECT 1 FROM outbox_event WHERE event_id = $1)
	`, pgtype.UUID{Bytes: eventID, Valid: true}, userID, ip, at)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "outbox_event_pkey" {
		// A concurrent poller already recorded the same login.
		return nil
	}
	return err
}
