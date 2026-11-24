package models

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var ErrLimitsNotConfigured = errors.New("transfer limits are not configured")

type TransferLimits struct {
	DailyLimit   decimal.Decimal
	MonthlyLimit decimal.Decimal
}

type TransferLimitUsage struct {
	Daily   LimitPeriodUsage
	Monthly LimitPeriodUsage
}

type LimitPeriodUsage struct {
	Limit    decimal.Decimal
	Used     decimal.Decimal
	ResetsAt time.Time
}

func (p LimitPeriodUsage) Remaining() decimal.Decimal {
	return decimal.Max(decimal.Zero, p.Limit.Sub(p.Used))
}
