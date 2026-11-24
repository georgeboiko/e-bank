package request

import (
	"encoding/json"
	"errors"
	"io"

	"e-bank/backend/internal/money/models"

	"github.com/shopspring/decimal"
)

func ParseTransferLimits(body io.Reader) (models.TransferLimits, error) {
	var payload struct {
		DailyLimit   string `json:"dailyLimit"`
		MonthlyLimit string `json:"monthlyLimit"`
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return models.TransferLimits{}, errors.New("проверьте поля и формат JSON")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return models.TransferLimits{}, errors.New("тело запроса должно содержать один JSON-объект")
	}

	if !positiveMoneyPattern.MatchString(payload.DailyLimit) || !positiveMoneyPattern.MatchString(payload.MonthlyLimit) {
		return models.TransferLimits{}, errors.New("dailyLimit и monthlyLimit должны быть строками с положительными суммами и двумя дробными знаками")
	}
	daily, dailyErr := decimal.NewFromString(payload.DailyLimit)
	monthly, monthlyErr := decimal.NewFromString(payload.MonthlyLimit)
	if dailyErr != nil || monthlyErr != nil {
		return models.TransferLimits{}, errors.New("некорректные значения лимитов")
	}
	return models.TransferLimits{DailyLimit: daily, MonthlyLimit: monthly}, nil
}
