package response

import (
	"time"

	"e-bank/backend/internal/antifraud/models"
)

type ReviewCase struct {
	ID         int32      `json:"id"`
	CheckID    int32      `json:"checkId"`
	TransferID int32      `json:"transferId"`
	AnalystID  *string    `json:"analystId"`
	Status     string     `json:"status"`
	Decision   *string    `json:"decision"`
	Comment    *string    `json:"comment"`
	CreatedAt  time.Time  `json:"createdAt"`
	DecidedAt  *time.Time `json:"decidedAt"`
}

func FromReviewCase(review models.ReviewCase) ReviewCase {
	decidedAt := review.DecidedAt
	if decidedAt != nil {
		utc := decidedAt.UTC()
		decidedAt = &utc
	}
	return ReviewCase{
		ID: review.ID, CheckID: review.CheckID, TransferID: review.TransferID,
		AnalystID: review.AnalystID, Status: review.Status, Decision: review.Decision, Comment: review.Comment,
		CreatedAt: review.CreatedAt.UTC(), DecidedAt: decidedAt,
	}
}
