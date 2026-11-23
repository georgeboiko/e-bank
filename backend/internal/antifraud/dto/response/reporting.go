package response

import (
	"encoding/json"
	"time"

	"e-bank/backend/internal/antifraud/models"
)

type Analytics struct {
	From                        time.Time        `json:"from"`
	To                          time.Time        `json:"to"`
	CheckCount                  int64            `json:"checkCount"`
	ChecksByAutomaticDecision   map[string]int64 `json:"checksByAutomaticDecision"`
	RiskScoreBuckets            []RiskBucket     `json:"riskScoreBuckets"`
	OpenReviewCount             int64            `json:"openReviewCount"`
	AverageProcessingMs         *float64         `json:"averageProcessingMs"`
	TriggeredRules              []RuleTriggers   `json:"triggeredRules"`
	TopRiskUsers                json.RawMessage  `json:"topRiskUsers"`
	SharedDevices               json.RawMessage  `json:"sharedDevices"`
	AccountsByCounterpartyCount json.RawMessage  `json:"accountsByCounterpartyCount"`
}

type RiskBucket struct {
	MinScore int   `json:"minScore"`
	MaxScore int   `json:"maxScore"`
	Count    int64 `json:"count"`
}

type RuleTriggers struct {
	RuleID int   `json:"ruleId"`
	Count  int64 `json:"count"`
}

func FromAnalytics(data models.Analytics) Analytics {
	triggers := make([]RuleTriggers, 0, 8)
	for index, count := range data.RuleTriggers {
		triggers = append(triggers, RuleTriggers{RuleID: index + 1, Count: count})
	}
	return Analytics{
		From: data.From.UTC(), To: data.To.UTC(),
		CheckCount: data.Approved + data.Review + data.Rejected,
		ChecksByAutomaticDecision: map[string]int64{
			"APPROVED": data.Approved, "REVIEW_REQUIRED": data.Review, "REJECTED": data.Rejected,
		},
		RiskScoreBuckets: []RiskBucket{
			{MinScore: 0, MaxScore: 29, Count: data.Approved},
			{MinScore: 30, MaxScore: 69, Count: data.Review},
			{MinScore: 70, MaxScore: 100, Count: data.Rejected},
		},
		OpenReviewCount: data.OpenReviews, AverageProcessingMs: data.AverageMilliseconds, TriggeredRules: triggers,
		TopRiskUsers: data.RiskUsers, SharedDevices: data.SharedDevices, AccountsByCounterpartyCount: data.AccountsByCounterpartyCount,
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

func ToAuditPage(events []models.AuditEvent, limit int, offset int64) AuditPage {
	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	items := make([]AuditEvent, 0, len(events))
	for _, event := range events {
		event.OccurredAt = event.OccurredAt.UTC()
		items = append(items, AuditEvent(event))
	}
	return AuditPage{Items: items, Limit: limit, Offset: offset, HasMore: hasMore}
}
