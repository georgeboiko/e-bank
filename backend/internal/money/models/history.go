package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type HistoryFilter struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int64
}

type TransferListFilter struct {
	HistoryFilter
	AccountID *int32
	UserID    *string
	State     string
	Type      string
	Direction string
	MinAmount *decimal.Decimal
	MaxAmount *decimal.Decimal
}

type LedgerEntry struct {
	ID         int32
	TransferID int32
	AccountID  int32
	Amount     decimal.Decimal
	CreatedAt  time.Time
}
