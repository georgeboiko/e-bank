package keycloak

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"e-bank/backend/internal/user/models"
)

var ErrUserNotFound = errors.New("Keycloak user not found")

type userRepresentation struct {
	ID                   string  `json:"id"`
	Username             string  `json:"username"`
	FirstName            *string `json:"firstName"`
	LastName             *string `json:"lastName"`
	Email                *string `json:"email"`
	Enabled              bool    `json:"enabled"`
	ServiceAccountClient string  `json:"serviceAccountClientId"`
}

func (c *Client) GetUser(ctx context.Context, id string) (models.User, error) {
	var user userRepresentation
	if err := c.do(ctx, http.MethodGet, []string{"users", id}, nil, nil, &user); err != nil {
		return models.User{}, userLookupError(err)
	}
	if user.isServiceAccount() {
		return models.User{}, ErrUserNotFound
	}
	if user.ID != id || user.Username == "" {
		return models.User{}, errors.New("invalid Keycloak user response")
	}
	return c.userWithRoles(ctx, user)
}

func (c *Client) ListUsers(ctx context.Context, filter models.UserFilter) ([]models.User, bool, error) {
	query := userQuery(filter.Enabled)
	if filter.Query != "" {
		query.Set("search", "*"+filter.Query+"*")
	}
	query.Set("briefRepresentation", "true")

	const pageSize = 100
	query.Set("max", strconv.Itoa(pageSize))
	var users []userRepresentation
	for first := 0; ; first += pageSize {
		query.Set("first", strconv.Itoa(first))
		var page []userRepresentation
		if err := c.do(ctx, http.MethodGet, []string{"users"}, query, nil, &page); err != nil {
			return nil, false, err
		}
		for _, user := range page {
			if user.isServiceAccount() {
				continue
			}
			if user.ID == "" || user.Username == "" {
				return nil, false, errors.New("invalid Keycloak user listing")
			}
			users = append(users, user)
		}
		if len(page) < pageSize {
			break
		}
	}

	slices.SortFunc(users, func(a, b userRepresentation) int {
		if order := cmp.Compare(a.Username, b.Username); order != 0 {
			return order
		}
		return cmp.Compare(a.ID, b.ID)
	})

	items := make([]models.User, 0, filter.Limit+1)
	skipped := 0
	for _, user := range users {
		if filter.Role == "" && skipped < filter.Offset {
			skipped++
			continue
		}
		mapped, err := c.userWithRoles(ctx, user)
		if errors.Is(err, ErrUserNotFound) {
			continue
		}
		if err != nil {
			return nil, false, err
		}
		if filter.Role != "" && !slices.Contains(mapped.Roles, filter.Role) {
			continue
		}
		if skipped < filter.Offset {
			skipped++
			continue
		}
		if len(items) == filter.Limit {
			return items, true, nil
		}
		items = append(items, mapped)
	}
	return items, false, nil
}

func (c *Client) CountUsers(ctx context.Context, enabled bool) (int64, error) {
	var count int64
	err := c.do(ctx, http.MethodGet, []string{"users", "count"}, userQuery(&enabled), nil, &count)
	return count, err
}

func userQuery(enabled *bool) url.Values {
	// The Keycloak search endpoint omits service accounts, including for a wildcard search.
	query := url.Values{"search": {"*"}}
	if enabled != nil {
		query.Set("enabled", strconv.FormatBool(*enabled))
	}
	return query
}

func (c *Client) userWithRoles(ctx context.Context, user userRepresentation) (models.User, error) {
	var mappings []struct {
		Name string `json:"name"`
	}
	err := c.do(ctx, http.MethodGet, []string{"users", user.ID, "role-mappings", "realm"}, nil, nil, &mappings)
	if err != nil {
		return models.User{}, userLookupError(err)
	}
	roles := make([]string, 0, 3)
	for _, role := range mappings {
		if models.IsApplicationRole(role.Name) {
			roles = append(roles, role.Name)
		}
	}
	slices.Sort(roles)
	return models.User{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Enabled:   user.Enabled,
		Roles:     slices.Compact(roles),
	}, nil
}

func (u userRepresentation) isServiceAccount() bool {
	return u.ServiceAccountClient != "" || strings.HasPrefix(u.Username, "service-account-")
}

func userLookupError(err error) error {
	var status *StatusError
	if errors.As(err, &status) && status.StatusCode == http.StatusNotFound {
		return ErrUserNotFound
	}
	return fmt.Errorf("read Keycloak user: %w", err)
}
