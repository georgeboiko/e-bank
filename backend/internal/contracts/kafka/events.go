package kafka

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/shopspring/decimal"
)

type EventType string

const (
	EventLoginSucceeded      EventType = "LoginSucceeded"
	EventFraudCheckRequested EventType = "FraudCheckRequested"
	EventFraudDecision       EventType = "FraudDecision"
)

type Decision string

const (
	DecisionApproved Decision = "APPROVED"
	DecisionRejected Decision = "REJECTED"
)

type DecisionSource string

const (
	SourceAutomatic DecisionSource = "AUTOMATIC"
	SourceAnalyst   DecisionSource = "ANALYST"
)

type Event[T any] struct {
	EventID    string    `json:"eventId"`
	EventType  EventType `json:"eventType"`
	OccurredAt time.Time `json:"occurredAt"`
	Data       T         `json:"data"`
}

type LoginSucceeded = Event[LoginSucceededData]
type FraudCheckRequested = Event[FraudCheckRequestedData]
type FraudDecision = Event[FraudDecisionData]

type LoginSucceededData struct {
	UserID    string     `json:"userId"`
	IPAddress netip.Addr `json:"ipAddress"`
}

type FraudCheckRequestedData struct {
	TransferID    int32           `json:"transferId"`
	FromAccountID int32           `json:"fromAccountId"`
	ToAccountID   int32           `json:"toAccountId"`
	SenderID      string          `json:"senderId"`
	RecipientID   string          `json:"recipientId"`
	Amount        decimal.Decimal `json:"amount"`
	DeviceID      *string         `json:"deviceId"`
	IPAddress     *netip.Addr     `json:"ipAddress"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type FraudDecisionData struct {
	TransferID     int32           `json:"transferId"`
	Decision       Decision        `json:"decision"`
	RiskScore      int             `json:"riskScore"`
	Source         DecisionSource  `json:"source"`
	TriggeredRules []TriggeredRule `json:"triggeredRules"`
}

type TriggeredRule struct {
	RuleID int    `json:"ruleId"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

func Decode[T any](payload []byte, expectedType EventType) (Event[T], error) {
	var event Event[T]
	if err := json.Unmarshal(payload, &event); err != nil {
		return Event[T]{}, fmt.Errorf("decode %s: %w", expectedType, err)
	}
	if event.EventType != expectedType {
		return Event[T]{}, fmt.Errorf("expected event type %q, got %q", expectedType, event.EventType)
	}
	return event, nil
}
