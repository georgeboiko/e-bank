package services

import (
	"context"
	"fmt"

	"e-bank/backend/internal/user/keycloak"
	"e-bank/backend/internal/user/models"
	"e-bank/backend/internal/user/repositories"
)

type AdminActionService struct {
	directory *keycloak.Client
	audit     *repositories.AdminAuditRepository
}

func NewAdminActionService(directory *keycloak.Client, audit *repositories.AdminAuditRepository) *AdminActionService {
	return &AdminActionService{directory: directory, audit: audit}
}

func (s *AdminActionService) SetStatus(ctx context.Context, actorID, userID string, enabled bool) (models.User, error) {
	action := "USER_BLOCKED"
	if enabled {
		action = "USER_UNBLOCKED"
	}
	details := map[string]any{"enabled": enabled}
	return s.perform(ctx, actorID, userID, action, details, func(user *models.User) error {
		if user.Enabled != enabled {
			if err := s.directory.SetUserEnabled(ctx, userID, enabled); err != nil {
				return err
			}
			user.Enabled = enabled
		}
		if !enabled {
			err := s.directory.LogoutUser(ctx, userID)
			details["sessionsTerminated"] = err == nil
			return err
		}
		return nil
	})
}

func (s *AdminActionService) SetRoles(ctx context.Context, actorID, userID string, roles []string) (models.User, error) {
	details := map[string]any{"roles": roles}
	return s.perform(ctx, actorID, userID, "USER_ROLES_CHANGED", details, func(user *models.User) error {
		details["previousRoles"] = user.Roles
		if err := s.directory.ReplaceUserRoles(ctx, userID, user.Roles, roles); err != nil {
			return err
		}
		user.Roles = roles
		return nil
	})
}

func (s *AdminActionService) TerminateSessions(ctx context.Context, actorID, userID string) error {
	_, err := s.perform(ctx, actorID, userID, "USER_SESSIONS_TERMINATED", map[string]any{}, func(_ *models.User) error {
		return s.directory.LogoutUser(ctx, userID)
	})
	return err
}

func (s *AdminActionService) perform(
	ctx context.Context,
	actorID, userID, action string,
	details map[string]any,
	update func(*models.User) error,
) (models.User, error) {
	var result models.User
	var updateErr error
	err := s.audit.WithUserLock(ctx, userID, func(audit *repositories.AdminAuditTx) error {
		user, err := s.directory.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		updateErr = update(&user)
		if updateErr != nil {
			details["result"] = "ERROR"
		} else {
			details["result"] = "SUCCESS"
		}
		result = user
		return audit.Record(ctx, actorID, userID, action, details)
	})
	if err != nil {
		return models.User{}, fmt.Errorf("persist user admin action: %w", err)
	}
	return result, updateErr
}
