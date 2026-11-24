package models

import (
	"encoding/json"
	"time"
)

type Analytics struct {
	From, To                  time.Time
	EnabledUsers              int64
	DisabledUsers             int64
	LoginCount                int64
	UniqueLoggedInUsers       int64
	AdministrativeActionCount int64
}

type AuditFilter struct {
	From, To *time.Time
	ActorID  *string
	UserID   *string
	Action   *string
	Limit    int
	Offset   int64
}

type AuditEvent struct {
	ID         int64
	OccurredAt time.Time
	ActorID    string
	Action     string
	UserID     string
	Details    json.RawMessage
}
