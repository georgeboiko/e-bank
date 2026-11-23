package services

import (
	"context"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportingService struct {
	pool *pgxpool.Pool
}

func NewReportingService(pool *pgxpool.Pool) *ReportingService {
	return &ReportingService{pool: pool}
}

func (s *ReportingService) Analytics(ctx context.Context, from, to time.Time, topLimit int) (models.Analytics, error) {
	return repositories.ReadAnalytics(ctx, s.pool, from, to, topLimit)
}

func (s *ReportingService) Audit(ctx context.Context, filter models.AuditFilter, forCheck bool) ([]models.AuditEvent, error) {
	if forCheck {
		exists, err := repositories.CheckExists(ctx, s.pool, *filter.CheckID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, models.ErrAuditCheckNotFound
		}
	}
	return repositories.ReadAudit(ctx, s.pool, filter)
}
