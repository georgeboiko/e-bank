package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type Analytics struct {
	From, To                      time.Time
	CreatedTransferCount          int64
	Transfers                     TransferStateCounts
	Accounts                      AccountStateCounts
	CompletedShare, RejectedShare float64
	Turnover, TopupTurnover       decimal.Decimal
}

type TransferStateCounts struct {
	Pending, FraudCheckPending, Completing, ReleasingFunds, Completed, Rejected, Cancelled int64
}

type AccountStateCounts struct {
	Active, Blocked, Closed int64
}

type AuditFilter struct {
	HistoryFilter
	ActorID   *string
	AccountID *int32
	Action    *string
}

type AuditEvent struct {
	ID         string
	OccurredAt time.Time
	ActorType  string
	ActorID    *string
	Action     string
	ObjectType string
	ObjectID   *string
	Details    json.RawMessage
}
