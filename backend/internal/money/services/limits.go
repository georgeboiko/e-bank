package services

import (
	"context"

	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"
)

type TransferLimitService struct {
	repository *repositories.TransferLimitRepository
}

func NewTransferLimitService(repository *repositories.TransferLimitRepository) *TransferLimitService {
	return &TransferLimitService{repository: repository}
}

func (s *TransferLimitService) Get(ctx context.Context) (models.TransferLimits, error) {
	return s.repository.Get(ctx)
}

func (s *TransferLimitService) Update(ctx context.Context, actorID string, limits models.TransferLimits) error {
	return s.repository.Update(ctx, actorID, limits)
}

func (s *TransferLimitService) GetUsage(ctx context.Context, ownerID string) (models.TransferLimitUsage, error) {
	return s.repository.GetUsage(ctx, ownerID)
}
