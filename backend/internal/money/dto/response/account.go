package response

import "time"

type Account struct {
	ID               int32     `json:"id"`
	OwnerID          string    `json:"ownerId"`
	Type             string    `json:"type"`
	State            string    `json:"state"`
	Balance          string    `json:"balance"`
	ReservedSum      string    `json:"reservedSum"`
	AvailableBalance string    `json:"availableBalance"`
	OpenedAt         time.Time `json:"openedAt"`
}
