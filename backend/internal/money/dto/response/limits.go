package response

import (
	"time"

	"e-bank/backend/internal/money/models"
)

type TransferLimits struct {
	DailyLimit   string `json:"dailyLimit"`
	MonthlyLimit string `json:"monthlyLimit"`
}

func FromTransferLimits(limits models.TransferLimits) TransferLimits {
	return TransferLimits{
		DailyLimit:   limits.DailyLimit.StringFixed(2),
		MonthlyLimit: limits.MonthlyLimit.StringFixed(2),
	}
}

type TransferLimitUsage struct {
	TimeZone string           `json:"timeZone"`
	Daily    LimitPeriodUsage `json:"daily"`
	Monthly  LimitPeriodUsage `json:"monthly"`
}

type LimitPeriodUsage struct {
	Limit     string    `json:"limit"`
	Used      string    `json:"used"`
	Remaining string    `json:"remaining"`
	ResetsAt  time.Time `json:"resetsAt"`
}

func FromTransferLimitUsage(usage models.TransferLimitUsage) TransferLimitUsage {
	return TransferLimitUsage{
		TimeZone: "Europe/Moscow",
		Daily:    fromLimitPeriodUsage(usage.Daily),
		Monthly:  fromLimitPeriodUsage(usage.Monthly),
	}
}

func fromLimitPeriodUsage(usage models.LimitPeriodUsage) LimitPeriodUsage {
	return LimitPeriodUsage{
		Limit:     usage.Limit.StringFixed(2),
		Used:      usage.Used.StringFixed(2),
		Remaining: usage.Remaining().StringFixed(2),
		ResetsAt:  usage.ResetsAt.UTC(),
	}
}
