package models

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrTopupNotFound        = errors.New("topup not found")
	ErrTopupsUnavailable    = errors.New("test payment provider is not configured")
	ErrTopupPaymentMismatch = errors.New("provider payment does not match the topup")
	ErrTopupRetryExpired    = errors.New("the provider idempotency window has expired")
)

type CreateTopup struct {
	ActorID        string
	AccountID      int32
	Amount         decimal.Decimal
	IdempotencyKey string
}

type Topup struct {
	ID              int32
	AccountID       int32
	Amount          decimal.Decimal
	State           string
	ConfirmationURL *string
	PaymentID       *string
	IdempotencyKey  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	FinishedAt      *time.Time
}
