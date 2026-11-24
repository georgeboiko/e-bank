package response

import (
	"encoding/json"
	"strconv"
	"time"

	"e-bank/backend/internal/user/models"
)

type Analytics struct {
	From                      time.Time `json:"from"`
	To                        time.Time `json:"to"`
	EnabledUsers              int64     `json:"enabledUsers"`
	DisabledUsers             int64     `json:"disabledUsers"`
	LoginCount                int64     `json:"loginCount"`
	UniqueLoggedInUsers       int64     `json:"uniqueLoggedInUsers"`
	AdministrativeActionCount int64     `json:"administrativeActionCount"`
}

func FromAnalytics(data models.Analytics) Analytics {
	return Analytics{
		From: data.From.UTC(), To: data.To.UTC(),
		EnabledUsers: data.EnabledUsers, DisabledUsers: data.DisabledUsers,
		LoginCount: data.LoginCount, UniqueLoggedInUsers: data.UniqueLoggedInUsers,
		AdministrativeActionCount: data.AdministrativeActionCount,
	}
}

type AuditEvent struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurredAt"`
	ActorType  string          `json:"actorType"`
	ActorID    string          `json:"actorId"`
	Action     string          `json:"action"`
	ObjectType string          `json:"objectType"`
	ObjectID   string          `json:"objectId"`
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
		items = append(items, AuditEvent{
			ID: strconv.FormatInt(event.ID, 10), OccurredAt: event.OccurredAt.UTC(),
			ActorType: "ADMIN", ActorID: event.ActorID,
			Action: event.Action, ObjectType: "USER", ObjectID: event.UserID, Details: event.Details,
		})
	}
	return AuditPage{Items: items, Limit: limit, Offset: offset, HasMore: hasMore}
}
