package services

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/user/models"
	"e-bank/backend/internal/user/repositories"
)

type UserCounter interface {
	CountUsers(ctx context.Context, enabled bool) (int64, error)
}

type ReportingService struct {
	repository *repositories.ReportingRepository
	users      UserCounter
}

func NewReportingService(repository *repositories.ReportingRepository, users UserCounter) *ReportingService {
	return &ReportingService{repository: repository, users: users}
}

func (s *ReportingService) Analytics(ctx context.Context, from, to time.Time) (models.Analytics, error) {
	data, err := s.repository.Analytics(ctx, from, to)
	if err != nil {
		return models.Analytics{}, fmt.Errorf("read User analytics: %w", err)
	}
	data.EnabledUsers, err = s.users.CountUsers(ctx, true)
	if err != nil {
		return models.Analytics{}, fmt.Errorf("count enabled users: %w", err)
	}
	data.DisabledUsers, err = s.users.CountUsers(ctx, false)
	if err != nil {
		return models.Analytics{}, fmt.Errorf("count disabled users: %w", err)
	}
	return data, nil
}

func (s *ReportingService) Audit(ctx context.Context, filter models.AuditFilter) ([]models.AuditEvent, bool, error) {
	events, err := s.repository.Audit(ctx, filter)
	if err != nil {
		return nil, false, fmt.Errorf("read User audit: %w", err)
	}
	hasMore := len(events) > filter.Limit
	if hasMore {
		events = events[:filter.Limit]
	}
	return events, hasMore, nil
}
