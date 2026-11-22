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

func (s *AccountService) ListAccounts(ctx context.Context, subject string, admin bool, filter models.AccountListFilter) ([]models.Account, bool, error) {
	if !admin {
		filter.OwnerID = &subject
	}
	accounts, err := s.repository.ListAccounts(ctx, filter.OwnerID, filter.State, filter.Limit+1, filter.Offset)
	if err != nil {
		return nil, false, fmt.Errorf("list accounts: %w", err)
	}
	hasMore := len(accounts) > filter.Limit
	if hasMore {
		accounts = accounts[:filter.Limit]
	}
	return accounts, hasMore, nil
}

func (s *AccountService) GetAccount(ctx context.Context, subject string, admin bool, id int32) (models.Account, error) {
	var ownerID *string
	if !admin {
		ownerID = &subject
	}
	return s.repository.GetAccount(ctx, id, ownerID)
}
