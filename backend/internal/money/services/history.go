package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/money/models"
)

func (s *TransferService) ListTransfers(ctx context.Context, subject string, admin bool, filter models.TransferListFilter) ([]models.Transfer, bool, error) {
	var ownerID *string
	if !admin {
		ownerID = &subject
	}
	limit := filter.Limit
	filter.Limit++
	transfers, err := s.repository.List(ctx, ownerID, filter)
	if err != nil {
		return nil, false, fmt.Errorf("list transfers: %w", err)
	}
	hasMore := len(transfers) > limit
	if hasMore {
		transfers = transfers[:limit]
	}
	return transfers, hasMore, nil
}

func (s *AccountService) ListLedger(ctx context.Context, subject string, admin bool, accountID int32, filter models.HistoryFilter) ([]models.LedgerEntry, bool, error) {
	var ownerID *string
	if !admin {
		ownerID = &subject
	}
	limit := filter.Limit
	filter.Limit++
	entries, err := s.repository.ListLedger(ctx, accountID, ownerID, filter)
	if err != nil {
		return nil, false, fmt.Errorf("list ledger entries: %w", err)
	}
	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}
	return entries, hasMore, nil
}
