package response

import "time"

type TransferPage struct {
	Items   []Transfer `json:"items"`
	Limit   int        `json:"limit"`
	Offset  int64      `json:"offset"`
	HasMore bool       `json:"hasMore"`
}

type LedgerEntry struct {
	ID         int32     `json:"id"`
	TransferID int32     `json:"transferId"`
	AccountID  int32     `json:"accountId"`
	Amount     string    `json:"amount"`
	CreatedAt  time.Time `json:"createdAt"`
}

type LedgerPage struct {
	Items   []LedgerEntry `json:"items"`
	Limit   int           `json:"limit"`
	Offset  int64         `json:"offset"`
	HasMore bool          `json:"hasMore"`
}
