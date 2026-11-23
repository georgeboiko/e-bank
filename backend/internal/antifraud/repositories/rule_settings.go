package repositories

import (
	"context"
	"fmt"

	"e-bank/backend/internal/antifraud/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ruleSettingsReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func ReadRuleConfigs(ctx context.Context, db ruleSettingsReader) (models.RuleConfigs, error) {
	configs := models.DefaultRuleConfigs()
	rows, err := db.Query(ctx, `SELECT rule_id, enabled, points, parameters FROM rule_settings`)
	if err != nil {
		return configs, fmt.Errorf("read rule settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int32
		var settings models.RuleSettings
		if err := rows.Scan(&id, &settings.Enabled, &settings.Points, &settings.Parameters); err != nil {
			return configs, fmt.Errorf("decode rule settings: %w", err)
		}
		configs[id-1].RuleSettings = settings
	}
	return configs, rows.Err()
}

func UpdateRuleSettings(ctx context.Context, pool *pgxpool.Pool, adminID string, rule models.RuleConfig, next models.RuleSettings) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO rule_settings (rule_id, enabled, points, parameters)
			VALUES ($1, $2, $3, $4) ON CONFLICT (rule_id) DO NOTHING
		`, rule.ID, rule.Enabled, rule.Points, rule.JSONParameters())
		if err != nil {
			return err
		}
		var before []byte
		err = tx.QueryRow(ctx, `
			SELECT jsonb_build_object('enabled', enabled, 'points', points, 'parameters', parameters)
			FROM rule_settings WHERE rule_id = $1 FOR UPDATE
		`, rule.ID).Scan(&before)
		if err != nil {
			return err
		}
		return updateRuleAndAudit(ctx, tx, adminID, rule.ID, next, before)
	})
}

func updateRuleAndAudit(ctx context.Context, tx pgx.Tx, adminID string, id int32, settings models.RuleSettings, before []byte) error {
	changed, err := tx.Exec(ctx, `
		UPDATE rule_settings SET enabled = $2, points = $3, parameters = $4
		WHERE rule_id = $1 AND (enabled, points, parameters) IS DISTINCT FROM ($2, $3, $4)
	`, id, settings.Enabled, settings.Points, settings.JSONParameters())
	if err != nil || changed.RowsAffected() == 0 {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app_user (id) VALUES ($1) ON CONFLICT DO NOTHING`, adminID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_log (occurred_at, actor_type, actor_id, action, details)
		SELECT clock_timestamp(), 'ADMIN', $2, 'RULE_UPDATED', jsonb_build_object('ruleId', rule_id, 'before', $3::jsonb,
		       'after', jsonb_build_object('enabled', enabled, 'points', points, 'parameters', parameters))
		FROM rule_settings WHERE rule_id = $1
	`, id, adminID, before)
	return err
}
