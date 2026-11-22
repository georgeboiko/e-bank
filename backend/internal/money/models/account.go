package models

import (
	"errors"
	"time"
)

var (
	ErrAccountLimitReached = errors.New("account limit reached")
	ErrAccountNotFound     = errors.New("account not found")
)

type Account struct {
	ID               int32
	OwnerID          *string
	Type             string
	State            string
	Balance          string
	ReservedSum      string
	AvailableBalance string
	OpenedAt         time.Time
}

type AccountListFilter struct {
	OwnerID *string
	State   string
	Limit   int
	Offset  int64
}
