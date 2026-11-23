package services

import (
	"context"
	"fmt"
	"time"

	"e-bank/backend/internal/infrastructure/yookassa"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"

	"github.com/shopspring/decimal"
)

type TopupProvider interface {
	CreatePayment(context.Context, string, decimal.Decimal) (yookassa.Payment, error)
	GetPayment(context.Context, string) (yookassa.Payment, error)
}

type TopupService struct {
	repository  *repositories.TopupRepository
	provider    TopupProvider
	technicalID int32
}

func NewTopupService(repository *repositories.TopupRepository, provider TopupProvider, technicalID int32) *TopupService {
	return &TopupService{repository: repository, provider: provider, technicalID: technicalID}
}

func (s *TopupService) Create(ctx context.Context, input models.CreateTopup) (models.Topup, bool, error) {
	if s.provider == nil {
		return models.Topup{}, false, models.ErrTopupsUnavailable
	}
	topup, created, err := s.repository.Create(ctx, s.technicalID, input)
	if err != nil {
		return models.Topup{}, false, fmt.Errorf("create topup: %w", err)
	}
	if topup.State != "PENDING" {
		return topup, created, nil
	}
	if topup.PaymentID != nil {
		payment, err := s.provider.GetPayment(ctx, *topup.PaymentID)
		if err != nil {
			return models.Topup{}, false, fmt.Errorf("read existing topup payment: %w", err)
		}
		if err := matchTopupPayment(topup, payment); err != nil {
			return models.Topup{}, false, err
		}
		topup.ConfirmationURL = payment.ConfirmationURL
		return topup, created, nil
	}

	// YooKassa deduplicates for 24 hours. Leave time for the request to reach the provider.
	if time.Since(topup.CreatedAt) >= 24*time.Hour-10*time.Second {
		return models.Topup{}, false, models.ErrTopupRetryExpired
	}
	payment, err := s.provider.CreatePayment(ctx, topup.IdempotencyKey, topup.Amount)
	if err != nil {
		return models.Topup{}, false, fmt.Errorf("create topup payment: %w", err)
	}
	if err := matchTopupPayment(topup, payment); err != nil {
		return models.Topup{}, false, err
	}
	if err := s.repository.RegisterPayment(ctx, topup.ID, payment.ID); err != nil {
		return models.Topup{}, false, fmt.Errorf("register topup payment: %w", err)
	}
	topup.ConfirmationURL = payment.ConfirmationURL
	return topup, created, nil
}

func matchTopupPayment(topup models.Topup, payment yookassa.Payment) error {
	if (topup.PaymentID != nil && payment.ID != *topup.PaymentID) ||
		payment.Currency != "RUB" || !payment.Amount.Equal(topup.Amount) {
		return models.ErrTopupPaymentMismatch
	}
	return nil
}
