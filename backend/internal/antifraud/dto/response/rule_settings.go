package response

import "e-bank/backend/internal/antifraud/models"

type RuleConfig struct {
	RuleID     int32            `json:"ruleId"`
	Name       string           `json:"name"`
	Enabled    bool             `json:"enabled"`
	Points     map[string]int32 `json:"points"`
	Parameters map[string]any   `json:"parameters"`
}

func ToRuleConfig(config models.RuleConfig) RuleConfig {
	return RuleConfig{RuleID: config.ID, Name: config.Name, Enabled: config.Enabled,
		Points: config.Points, Parameters: config.JSONParameters()}
}
