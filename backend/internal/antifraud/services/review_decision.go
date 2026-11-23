package services

import (
	"context"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
)

type ReviewDecisionService struct {
	repository *repositories.ReviewDecisionRepository
}

func NewReviewDecisionService(repository *repositories.ReviewDecisionRepository) *ReviewDecisionService {
	return &ReviewDecisionService{repository: repository}
}

func (s *ReviewDecisionService) Claim(ctx context.Context, caseID int32, analystID string) (models.ReviewCase, error) {
	return s.repository.Claim(ctx, caseID, analystID)
}

func (s *ReviewDecisionService) Resolve(ctx context.Context, caseID int32, analystID, decision string, comment *string) (models.ReviewCase, error) {
	return s.repository.Resolve(ctx, caseID, analystID, decision, comment)
}
