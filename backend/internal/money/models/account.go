package models

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrAccountLimitReached  = errors.New("account limit reached")
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountStateConflict = errors.New("account state conflict")
)

type Account struct {
	ID               int32
	OwnerID          *string
	Type             string
	State            string
	Balance          decimal.Decimal
	ReservedSum      decimal.Decimal
	AvailableBalance decimal.Decimal
	OpenedAt         time.Time
}

type AccountListFilter struct {
	OwnerID *string
	State   string
	Limit   int
	Offset  int64
}
