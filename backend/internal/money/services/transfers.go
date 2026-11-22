package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"
)

type TransferService struct {
	repository *repositories.TransferRepository
}

func NewTransferService(repository *repositories.TransferRepository) *TransferService {
	return &TransferService{repository: repository}
}

func (s *TransferService) CreateTransfer(ctx context.Context, input models.CreateTransfer) (models.Transfer, bool, error) {
	transfer, created, err := s.repository.Create(ctx, input)
	if err != nil {
		return models.Transfer{}, false, fmt.Errorf("create transfer: %w", err)
	}
	return transfer, created, nil
}

func (s *TransferService) GetTransfer(ctx context.Context, subject string, admin bool, id int32) (models.Transfer, error) {
	var ownerID *string
	if !admin {
		ownerID = &subject
	}
	return s.repository.Get(ctx, id, ownerID)
}
