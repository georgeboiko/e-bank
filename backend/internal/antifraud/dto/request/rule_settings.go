package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"

	"e-bank/backend/internal/antifraud/models"

	"github.com/shopspring/decimal"
)

var ruleMoneyPattern = regexp.MustCompile(`^(0\.(0[1-9]|[1-9][0-9])|[1-9][0-9]{0,16}\.[0-9]{2})$`)

func ParseRuleSettings(r io.Reader, rule models.RuleConfig) (models.RuleSettings, error) {
	var input struct {
		Enabled    *bool                      `json:"enabled"`
		Points     map[string]int32           `json:"points"`
		Parameters map[string]json.RawMessage `json:"parameters"`
	}
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return models.RuleSettings{}, errors.New("Некорректный JSON настройки правила.")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return models.RuleSettings{}, errors.New("Ожидается один JSON-объект.")
	}
	if input.Enabled == nil || len(input.Points) != len(rule.Points) || len(input.Parameters) != len(rule.Parameters) {
		return models.RuleSettings{}, errors.New("Нужны enabled, points и все параметры выбранного правила.")
	}
	for key := range rule.Points {
		if input.Points[key] < 1 || input.Points[key] > 100 {
			return models.RuleSettings{}, fmt.Errorf("Поле points.%s должно быть от 1 до 100.", key)
		}
	}
	if input.Points["weak"] > input.Points["strong"] {
		return models.RuleSettings{}, errors.New("Слабый сигнал не может давать больше баллов, чем сильный.")
	}

	parameters := make(map[string]decimal.Decimal, len(rule.Parameters))
	for name := range rule.Parameters {
		value, err := parseRuleParameter(name, input.Parameters[name])
		if err != nil {
			return models.RuleSettings{}, err
		}
		parameters[name] = value
	}
	switch rule.ID {
	case 1:
		if parameters["longWindowSeconds"].LessThan(parameters["shortWindowSeconds"]) ||
			parameters["longCount"].LessThan(parameters["shortCount"]) {
			return models.RuleSettings{}, errors.New("Длинное окно и его порог не могут быть меньше короткого.")
		}
	case 2:
		for _, suffix := range []string{"MinAmount", "MedianFactor", "MadFactor"} {
			if parameters["strong"+suffix].LessThan(parameters["weak"+suffix]) {
				return models.RuleSettings{}, errors.New("Параметры сильного порога не могут быть меньше слабых.")
			}
		}
	}
	return models.RuleSettings{Enabled: *input.Enabled, Points: input.Points, Parameters: parameters}, nil
}

func parseRuleParameter(name string, raw json.RawMessage) (decimal.Decimal, error) {
	invalid := fmt.Errorf("Некорректное значение parameters.%s.", name)
	if len(raw) == 0 || len(raw) > 100 {
		return decimal.Zero, invalid
	}
	if name == "weakMinAmount" || name == "strongMinAmount" {
		var amount string
		if err := json.Unmarshal(raw, &amount); err != nil || !ruleMoneyPattern.MatchString(amount) {
			return decimal.Zero, invalid
		}
		return decimal.NewFromString(amount)
	}
	if raw[0] != '-' && (raw[0] < '0' || raw[0] > '9') {
		return decimal.Zero, invalid
	}
	value, err := decimal.NewFromString(string(raw))
	if err != nil || value.IsNegative() || value.Exponent() < -308 || value.Exponent() > 308 {
		return decimal.Zero, invalid
	}
	switch name {
	case "weakMedianFactor", "weakMadFactor", "strongMedianFactor", "strongMadFactor":
		return value, nil
	case "maxAmountRatio":
		if value.GreaterThanOrEqual(decimal.NewFromInt(1)) {
			return value, nil
		}
	case "minDistanceKm":
		if value.IsPositive() {
			return value, nil
		}
	default:
		if value.IsInteger() && value.IsPositive() && value.LessThanOrEqual(decimal.NewFromInt(2147483647)) {
			return value, nil
		}
	}
	return decimal.Zero, invalid
}
