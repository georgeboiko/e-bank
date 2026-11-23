package models

import (
	"errors"
	"time"
)

var ErrCheckNotFound = errors.New("antifraud check not found")

type CheckListFilter struct {
	TransferID        *int32
	UserID            *string
	FromAccountID     *int32
	ToAccountID       *int32
	AutomaticDecision string
	MinScore          *int32
	MaxScore          *int32
	From              *time.Time
	To                *time.Time
	Limit             int
	Offset            int64
}

type ReviewCaseListFilter struct {
	Status    string
	AnalystID *string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int64
}

type CheckSummary struct {
	ID                int32
	TransferID        int32
	RiskScore         int32
	AutomaticDecision string
	FinalDecision     *string
	DecisionSource    *string
	ReviewCaseID      *int32
	CheckedAt         time.Time
}

type Check struct {
	CheckSummary
	Transfer TransferSnapshot
	Rules    []RuleEvaluation
}

type TransferSnapshot struct {
	ID            int32
	FromAccountID int32
	ToAccountID   int32
	SenderID      string
	RecipientID   string
	Amount        string
	DeviceID      *string
	IPAddress     *string
	CreatedAt     time.Time
	ReceivedAt    time.Time
	Geo           Geo
}
