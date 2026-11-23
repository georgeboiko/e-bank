package models

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type RuleSettings struct {
	Enabled    bool
	Points     map[string]int32
	Parameters map[string]decimal.Decimal
}

type RuleConfig struct {
	ID   int32
	Name string
	RuleSettings
}

type RuleConfigs [8]RuleConfig

func (s RuleSettings) Integer(name string) int64 {
	return s.Parameters[name].IntPart()
}

func (s RuleSettings) JSONParameters() map[string]any {
	result := make(map[string]any, len(s.Parameters))
	for key, value := range s.Parameters {
		switch key {
		case "weakMinAmount", "strongMinAmount":
			result[key] = value.StringFixed(2)
		default:
			result[key] = json.Number(value.String())
		}
	}
	return result
}

func DefaultRuleConfigs() RuleConfigs {
	return RuleConfigs{
		{1, "Частые попытки", defaultRuleSettings(map[string]int32{"weak": 15, "strong": 30},
			map[string]int64{"shortWindowSeconds": 120, "shortCount": 3, "longWindowSeconds": 600, "longCount": 6})},
		{2, "Необычная сумма", defaultRuleSettings(map[string]int32{"weak": 15, "strong": 30},
			map[string]int64{"historyDays": 30, "weakMinAmount": 2000, "weakMedianFactor": 3, "weakMadFactor": 4,
				"strongMinAmount": 10000, "strongMedianFactor": 4, "strongMadFactor": 6})},
		{3, "Недавнее устройство", defaultRuleSettings(map[string]int32{"triggered": 15},
			map[string]int64{"newDeviceWindowSeconds": 86400})},
		{4, "Много новых получателей", defaultRuleSettings(map[string]int32{"triggered": 25},
			map[string]int64{"windowSeconds": 600, "minNewRecipients": 6})},
		{5, "Много отправителей", defaultRuleSettings(map[string]int32{"triggered": 30},
			map[string]int64{"windowSeconds": 3600, "minSenders": 6})},
		{6, "Общее устройство", defaultRuleSettings(map[string]int32{"triggered": 20},
			map[string]int64{"windowSeconds": 86400, "minUsers": 4})},
		{7, "Замыкание цикла", RuleSettings{Enabled: true, Points: map[string]int32{"triggered": 20},
			Parameters: map[string]decimal.Decimal{"windowSeconds": decimal.NewFromInt(3600), "maxAmountRatio": decimal.New(15, -1)}}},
		{8, "Резкая смена геолокации", defaultRuleSettings(map[string]int32{"triggered": 15},
			map[string]int64{"windowSeconds": 7200, "minDistanceKm": 500})},
	}
}

func defaultRuleSettings(points map[string]int32, values map[string]int64) RuleSettings {
	parameters := make(map[string]decimal.Decimal, len(values))
	for key, value := range values {
		parameters[key] = decimal.NewFromInt(value)
	}
	return RuleSettings{Enabled: true, Points: points, Parameters: parameters}
}
