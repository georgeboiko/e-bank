package models

import (
	"errors"
	"time"
)

var ErrAccountLimitReached = errors.New("account limit reached")

type Account struct {
	ID               int32
	OwnerID          string
	Type             string
	State            string
	Balance          string
	ReservedSum      string
	AvailableBalance string
	OpenedAt         time.Time
}
