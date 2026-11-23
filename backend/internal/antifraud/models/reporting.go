package models

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrAuditCheckNotFound = errors.New("antifraud check not found")

type Analytics struct {
	From, To                    time.Time
	Approved, Review, Rejected  int64
	OpenReviews                 int64
	AverageMilliseconds         *float64
	RuleTriggers                []int64
	RiskUsers, SharedDevices    json.RawMessage
	AccountsByCounterpartyCount json.RawMessage
}

type AuditFilter struct {
	From, To *time.Time
	ActorID  *string
	CheckID  *int32
	Action   *string
	Limit    int
	Offset   int64
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
