package services

import (
	"context"
	"errors"
	"strings"

	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/contracts/kafka"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func applyLoginSucceeded(ctx context.Context, pool *pgxpool.Pool, resolver GeoResolver, record *kgo.Record) error {
	event, err := kafka.Decode[kafka.LoginSucceededData](record.Value, kafka.EventLoginSucceeded)
	if err != nil {
		return err
	}
	if event.OccurredAt.IsZero() {
		return errors.New("LoginSucceeded requires occurredAt")
	}
	if strings.TrimSpace(event.Data.UserID) == "" || strings.ContainsRune(event.Data.UserID, '\x00') {
		return errors.New("LoginSucceeded requires a valid userId")
	}
	if !event.Data.IPAddress.IsValid() || event.Data.IPAddress.Zone() != "" {
		return errors.New("LoginSucceeded ipAddress must be an IPv4 or IPv6 address")
	}
	geo := resolver.Resolve(ctx, event.Data.IPAddress)
	return repositories.StoreLogin(ctx, pool, event, record.Value, geo)
}
