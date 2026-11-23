package response

import (
	"time"

	"e-bank/backend/internal/money/models"
)

type Topup struct {
	ID              int32      `json:"id"`
	AccountID       int32      `json:"accountId"`
	Amount          string     `json:"amount"`
	State           string     `json:"state"`
	ConfirmationURL *string    `json:"confirmationUrl"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
}

func FromTopup(topup models.Topup) Topup {
	var finishedAt *time.Time
	if topup.FinishedAt != nil {
		at := topup.FinishedAt.UTC()
		finishedAt = &at
	}
	return Topup{ID: topup.ID, AccountID: topup.AccountID, Amount: topup.Amount.StringFixed(2), State: topup.State,
		ConfirmationURL: topup.ConfirmationURL, CreatedAt: topup.CreatedAt.UTC(), UpdatedAt: topup.UpdatedAt.UTC(),
		FinishedAt: finishedAt}
}
