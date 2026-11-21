package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"
)

type AccountService struct {
	repository           *repositories.AccountRepository
	maxAccountsPerClient int64
}

func NewAccountService(repository *repositories.AccountRepository, maxAccountsPerClient int64) *AccountService {
	return &AccountService{
		repository:           repository,
		maxAccountsPerClient: maxAccountsPerClient,
	}
}

func (s *AccountService) OpenAccount(ctx context.Context, ownerID string) (models.Account, error) {
	var account models.Account
	err := s.repository.WithOwnerLock(ctx, ownerID, func(accounts *repositories.AccountTx) error {
		count, err := accounts.Count(ctx)
		if err != nil {
			return err
		}
		if count >= s.maxAccountsPerClient {
			return models.ErrAccountLimitReached
		}

		account, err = accounts.Create(ctx)
		return err
	})
	if err != nil {
		return models.Account{}, fmt.Errorf("open client account: %w", err)
	}

	return account, nil
}
