package keycloak

import (
	"context"
	"errors"
	"net/http"
	"slices"
)

var ErrUserConflict = errors.New("user update conflicts with Keycloak state")

type realmRoleMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) SetUserEnabled(ctx context.Context, id string, enabled bool) error {
	payload := struct {
		Enabled bool `json:"enabled"`
	}{Enabled: enabled}
	err := c.do(ctx, http.MethodPut, []string{"users", id}, nil, payload, nil)
	return adminActionError(err)
}

func (c *Client) LogoutUser(ctx context.Context, id string) error {
	err := c.do(ctx, http.MethodPost, []string{"users", id, "logout"}, nil, nil, nil)
	return adminActionError(err)
}

func (c *Client) ReplaceUserRoles(ctx context.Context, id string, current, desired []string) error {
	path := []string{"users", id, "role-mappings", "realm"}
	var added, removed []realmRoleMapping

	for _, role := range desired {
		if !slices.Contains(current, role) {
			added = append(added, realmRoleMapping{Name: role})
		}
	}
	for _, role := range current {
		if !slices.Contains(desired, role) {
			removed = append(removed, realmRoleMapping{Name: role})
		}
	}
	if len(added) > 0 {
		if err := c.resolveRealmRoleMappings(ctx, append(slices.Clone(path), "available"), added); err != nil {
			return err
		}
	}
	if len(removed) > 0 {
		if err := c.resolveRealmRoleMappings(ctx, path, removed); err != nil {
			return err
		}
		if err := c.do(ctx, http.MethodDelete, path, nil, removed, nil); err != nil {
			return adminActionError(err)
		}
	}
	if len(added) > 0 {
		return adminActionError(c.do(ctx, http.MethodPost, path, nil, added, nil))
	}
	return nil
}

func (c *Client) resolveRealmRoleMappings(ctx context.Context, path []string, roles []realmRoleMapping) error {
	var available []realmRoleMapping
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &available); err != nil {
		return adminActionError(err)
	}
	for i := range roles {
		match := slices.IndexFunc(available, func(role realmRoleMapping) bool {
			return role.Name == roles[i].Name
		})
		if match == -1 || available[match].ID == "" {
			return ErrUserConflict
		}
		roles[i].ID = available[match].ID
	}
	return nil
}

func adminActionError(err error) error {
	var status *StatusError
	if errors.As(err, &status) {
		switch status.StatusCode {
		case http.StatusNotFound:
			return ErrUserNotFound
		case http.StatusConflict:
			return ErrUserConflict
		}
	}
	return err
}
