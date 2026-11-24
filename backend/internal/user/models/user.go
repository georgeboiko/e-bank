package models

type User struct {
	ID        string
	Username  string
	FirstName *string
	LastName  *string
	Email     *string
	Enabled   bool
	Roles     []string
}

type UserFilter struct {
	Query   string
	Enabled *bool
	Role    string
	Limit   int
	Offset  int
}

func IsApplicationRole(role string) bool {
	switch role {
	case "CLIENT", "ANALYST", "ADMIN":
		return true
	default:
		return false
	}
}
