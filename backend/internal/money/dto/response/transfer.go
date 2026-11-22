package response

import "time"

type Transfer struct {
	ID            int32      `json:"id"`
	Type          string     `json:"type"`
	FromAccountID int32      `json:"fromAccountId"`
	ToAccountID   int32      `json:"toAccountId"`
	Amount        string     `json:"amount"`
	State         string     `json:"state"`
	RejectCause   *string    `json:"rejectCause"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
}
