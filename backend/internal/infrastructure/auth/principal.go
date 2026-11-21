package auth

import (
	"context"
	"slices"
	"strings"
)

type Principal struct {
	Subject string
	Roles   []string
}

type principalKey struct{}

func WithVerifiedPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok && strings.TrimSpace(principal.Subject) != ""
}

func (p Principal) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}
