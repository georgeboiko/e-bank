package models

import (
	"errors"
	"net/netip"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrInvalidTransfer     = errors.New("invalid transfer")
	ErrIdempotencyConflict = errors.New("idempotency key already used for a different transfer")
	ErrInactiveAccount     = errors.New("transfer account is inactive")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrTransferLimit       = errors.New("transfer limit exceeded")
	ErrTransferNotFound    = errors.New("transfer not found")
)

type CreateTransfer struct {
	ActorID        string
	FromAccountID  int32
	ToAccountID    int32
	Amount         decimal.Decimal
	IdempotencyKey string
	DeviceID       *string
	IPAddress      netip.Addr
}

type Transfer struct {
	ID            int32
	Type          string
	FromAccountID int32
	ToAccountID   int32
	Amount        decimal.Decimal
	State         string
	RejectCause   *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	FinishedAt    *time.Time
}
