package services

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"
)

type ReportingService struct {
	repository *repositories.ReportingRepository
}

func NewReportingService(repository *repositories.ReportingRepository) *ReportingService {
	return &ReportingService{repository: repository}
}

func (s *ReportingService) Analytics(ctx context.Context, from, to time.Time) (models.Analytics, error) {
	data, err := s.repository.Analytics(ctx, from, to)
	if err != nil {
		return models.Analytics{}, fmt.Errorf("read Money analytics: %w", err)
	}
	if data.CreatedTransferCount > 0 {
		count := float64(data.CreatedTransferCount)
		data.CompletedShare = float64(data.Transfers.Completed) / count
		data.RejectedShare = float64(data.Transfers.Rejected) / count
	}
	return data, nil
}

func (s *ReportingService) Audit(ctx context.Context, filter models.AuditFilter) ([]models.AuditEvent, bool, error) {
	events, err := s.repository.Audit(ctx, filter)
	if err != nil {
		return nil, false, fmt.Errorf("read Money audit: %w", err)
	}
	hasMore := len(events) > filter.Limit
	if hasMore {
		events = events[:filter.Limit]
	}
	return events, hasMore, nil
}
