package response

import (
	"encoding/json"
	"time"

	"e-bank/backend/internal/money/models"
)

type Analytics struct {
	From                 time.Time        `json:"from"`
	To                   time.Time        `json:"to"`
	CreatedTransferCount int64            `json:"createdTransferCount"`
	TransfersByState     map[string]int64 `json:"transfersByState"`
	CompletedShare       float64          `json:"completedShare"`
	RejectedShare        float64          `json:"rejectedShare"`
	Turnover             string           `json:"turnover"`
	TopupTurnover        string           `json:"topupTurnover"`
	AccountsByState      map[string]int64 `json:"accountsByState"`
}

func FromAnalytics(data models.Analytics) Analytics {
	return Analytics{
		From:                 data.From.UTC(),
		To:                   data.To.UTC(),
		CreatedTransferCount: data.CreatedTransferCount,
		TransfersByState: map[string]int64{
			"PENDING":             data.Transfers.Pending,
			"FRAUD_CHECK_PENDING": data.Transfers.FraudCheckPending,
			"COMPLETING":          data.Transfers.Completing,
			"RELEASING_FUNDS":     data.Transfers.ReleasingFunds,
			"COMPLETED":           data.Transfers.Completed,
			"REJECTED":            data.Transfers.Rejected,
			"CANCELLED":           data.Transfers.Cancelled,
		},
		CompletedShare: data.CompletedShare,
		RejectedShare:  data.RejectedShare,
		Turnover:       data.Turnover.StringFixed(2),
		TopupTurnover:  data.TopupTurnover.StringFixed(2),
		AccountsByState: map[string]int64{
			"ACTIVE": data.Accounts.Active, "BLOCKED": data.Accounts.Blocked, "CLOSED": data.Accounts.Closed,
		},
	}
}

type AuditEvent struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurredAt"`
	ActorType  string          `json:"actorType"`
	ActorID    *string         `json:"actorId"`
	Action     string          `json:"action"`
	ObjectType string          `json:"objectType"`
	ObjectID   *string         `json:"objectId"`
	Details    json.RawMessage `json:"details"`
}

type AuditPage struct {
	Items   []AuditEvent `json:"items"`
	Limit   int          `json:"limit"`
	Offset  int64        `json:"offset"`
	HasMore bool         `json:"hasMore"`
}

func ToAuditPage(events []models.AuditEvent, limit int, offset int64, hasMore bool) AuditPage {
	items := make([]AuditEvent, 0, len(events))
	for _, event := range events {
		event.OccurredAt = event.OccurredAt.UTC()
		items = append(items, AuditEvent(event))
	}
	return AuditPage{Items: items, Limit: limit, Offset: offset, HasMore: hasMore}
}
