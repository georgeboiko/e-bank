package response

import "e-bank/backend/internal/user/models"

type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	FirstName *string  `json:"firstName"`
	LastName  *string  `json:"lastName"`
	Email     *string  `json:"email"`
	Enabled   bool     `json:"enabled"`
	Roles     []string `json:"roles"`
}

type UserPage struct {
	Items   []User `json:"items"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"hasMore"`
}

func FromUser(user models.User) User {
	return User(user)
}

func ToUserPage(users []models.User, filter models.UserFilter, hasMore bool) UserPage {
	items := make([]User, 0, len(users))
	for _, user := range users {
		items = append(items, FromUser(user))
	}
	return UserPage{Items: items, Limit: filter.Limit, Offset: filter.Offset, HasMore: hasMore}
}
