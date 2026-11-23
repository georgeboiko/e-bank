package models

import (
	"errors"
	"time"
)

var ErrReviewCaseNotFound = errors.New("review case not found")

type ReviewCase struct {
	ID         int32
	CheckID    int32
	TransferID int32
	AnalystID  *string
	Status     string
	Decision   *string
	Comment    *string
	CreatedAt  time.Time
	DecidedAt  *time.Time
}
