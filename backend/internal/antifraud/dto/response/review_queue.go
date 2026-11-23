package response

import (
	"time"

	"e-bank/backend/internal/antifraud/models"
)

type CheckSummary struct {
	ID                int32     `json:"id"`
	TransferID        int32     `json:"transferId"`
	RiskScore         int32     `json:"riskScore"`
	AutomaticDecision string    `json:"automaticDecision"`
	FinalDecision     *string   `json:"finalDecision"`
	DecisionSource    *string   `json:"decisionSource"`
	ReviewCaseID      *int32    `json:"reviewCaseId"`
	CheckedAt         time.Time `json:"checkedAt"`
}

type Check struct {
	CheckSummary
	Transfer TransferSnapshot `json:"transfer"`
	Rules    []RuleResult     `json:"rules"`
}

type TransferSnapshot struct {
	ID            int32     `json:"id"`
	FromAccountID int32     `json:"fromAccountId"`
	ToAccountID   int32     `json:"toAccountId"`
	SenderID      string    `json:"senderId"`
	RecipientID   string    `json:"recipientId"`
	Amount        string    `json:"amount"`
	DeviceID      *string   `json:"deviceId"`
	IPAddress     *string   `json:"ipAddress"`
	CreatedAt     time.Time `json:"createdAt"`
	ReceivedAt    time.Time `json:"receivedAt"`
	Geo           Geo       `json:"geo"`
}

type Geo struct {
	Status     string     `json:"status"`
	Latitude   *float64   `json:"latitude"`
	Longitude  *float64   `json:"longitude"`
	ResolvedAt *time.Time `json:"resolvedAt"`
}

type RuleResult struct {
	RuleID int32  `json:"ruleId"`
	Status string `json:"status"`
	Score  int32  `json:"score"`
	Reason string `json:"reason"`
}

type CheckPage struct {
	Items   []CheckSummary `json:"items"`
	Limit   int            `json:"limit"`
	Offset  int64          `json:"offset"`
	HasMore bool           `json:"hasMore"`
}

type ReviewPage struct {
	Items   []ReviewCase `json:"items"`
	Limit   int          `json:"limit"`
	Offset  int64        `json:"offset"`
	HasMore bool         `json:"hasMore"`
}

func FromCheckSummary(check models.CheckSummary) CheckSummary {
	return CheckSummary{
		ID: check.ID, TransferID: check.TransferID, RiskScore: check.RiskScore,
		AutomaticDecision: check.AutomaticDecision, FinalDecision: check.FinalDecision,
		DecisionSource: check.DecisionSource, ReviewCaseID: check.ReviewCaseID, CheckedAt: check.CheckedAt.UTC(),
	}
}

func FromCheck(check models.Check) Check {
	t := check.Transfer
	resolvedAt := t.Geo.ResolvedAt
	if resolvedAt != nil {
		utc := resolvedAt.UTC()
		resolvedAt = &utc
	}
	rules := make([]RuleResult, 0, len(check.Rules))
	for _, rule := range check.Rules {
		rules = append(rules, RuleResult{RuleID: rule.RuleID, Status: rule.Status, Score: rule.Score, Reason: rule.Reason})
	}
	return Check{
		CheckSummary: FromCheckSummary(check.CheckSummary),
		Transfer: TransferSnapshot{
			ID: t.ID, FromAccountID: t.FromAccountID, ToAccountID: t.ToAccountID,
			SenderID: t.SenderID, RecipientID: t.RecipientID, Amount: t.Amount,
			DeviceID: t.DeviceID, IPAddress: t.IPAddress, CreatedAt: t.CreatedAt.UTC(), ReceivedAt: t.ReceivedAt.UTC(),
			Geo: Geo{
				Status: t.Geo.Status, Latitude: t.Geo.Latitude, Longitude: t.Geo.Longitude,
				ResolvedAt: resolvedAt,
			},
		},
		Rules: rules,
	}
}
