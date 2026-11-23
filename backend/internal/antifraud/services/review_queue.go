package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
)

type ReviewQueueService struct {
	repository *repositories.ReviewQueueRepository
}

func NewReviewQueueService(repository *repositories.ReviewQueueRepository) *ReviewQueueService {
	return &ReviewQueueService{repository: repository}
}

func (s *ReviewQueueService) ListChecks(ctx context.Context, filter models.CheckListFilter) ([]models.CheckSummary, bool, error) {
	checks, err := s.repository.ListChecks(ctx, filter)
	if err != nil {
		return nil, false, fmt.Errorf("list antifraud checks: %w", err)
	}
	hasMore := len(checks) > filter.Limit
	if hasMore {
		checks = checks[:filter.Limit]
	}
	return checks, hasMore, nil
}

func (s *ReviewQueueService) GetCheck(ctx context.Context, id int32) (models.Check, error) {
	check, err := s.repository.GetCheck(ctx, id)
	if err != nil {
		return models.Check{}, fmt.Errorf("get antifraud check: %w", err)
	}
	check.Rules, err = s.repository.ListCheckRules(ctx, id)
	if err != nil {
		return models.Check{}, fmt.Errorf("list antifraud check rules: %w", err)
	}
	return check, nil
}

func (s *ReviewQueueService) ListReviewCases(ctx context.Context, filter models.ReviewCaseListFilter) ([]models.ReviewCase, bool, error) {
	reviews, err := s.repository.ListReviewCases(ctx, filter)
	if err != nil {
		return nil, false, fmt.Errorf("list antifraud review cases: %w", err)
	}
	hasMore := len(reviews) > filter.Limit
	if hasMore {
		reviews = reviews[:filter.Limit]
	}
	return reviews, hasMore, nil
}

func (s *ReviewQueueService) GetReviewCase(ctx context.Context, id int32) (models.ReviewCase, error) {
	review, err := s.repository.GetReviewCase(ctx, id)
	if err != nil {
		return models.ReviewCase{}, fmt.Errorf("get antifraud review case: %w", err)
	}
	return review, nil
}
