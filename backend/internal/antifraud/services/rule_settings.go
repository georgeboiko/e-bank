package services

import (
	"context"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleSettingsService struct {
	pool *pgxpool.Pool
}

func NewRuleSettingsService(pool *pgxpool.Pool) *RuleSettingsService {
	return &RuleSettingsService{pool: pool}
}

func (s *RuleSettingsService) List(ctx context.Context) (models.RuleConfigs, error) {
	return repositories.ReadRuleConfigs(ctx, s.pool)
}

func (s *RuleSettingsService) Update(ctx context.Context, adminID string, rule models.RuleConfig, next models.RuleSettings) (models.RuleConfig, error) {
	if err := repositories.UpdateRuleSettings(ctx, s.pool, adminID, rule, next); err != nil {
		return models.RuleConfig{}, err
	}
	rule.RuleSettings = next
	return rule, nil
}
