package request

import (
	"encoding/json"
	"errors"
	"io"

	"e-bank/backend/internal/money/models"

	"github.com/shopspring/decimal"
)

func ParseCreateTopup(body io.Reader, keys []string) (models.CreateTopup, error) {
	if len(keys) != 1 || !uuidPattern.MatchString(keys[0]) {
		return models.CreateTopup{}, errors.New("заголовок Idempotency-Key должен содержать один UUID")
	}
	var payload struct {
		AccountID int32  `json:"accountId"`
		Amount    string `json:"amount"`
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return models.CreateTopup{}, errors.New("проверьте поля и формат JSON")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return models.CreateTopup{}, errors.New("тело запроса должно содержать один JSON-объект")
	}
	if payload.AccountID <= 0 {
		return models.CreateTopup{}, errors.New("accountId должен быть положительным")
	}
	if !positiveMoneyPattern.MatchString(payload.Amount) {
		return models.CreateTopup{}, errors.New("amount должен быть строкой с положительной суммой и двумя дробными знаками")
	}
	amount, err := decimal.NewFromString(payload.Amount)
	if err != nil {
		return models.CreateTopup{}, errors.New("некорректная сумма amount")
	}
	return models.CreateTopup{AccountID: payload.AccountID, Amount: amount, IdempotencyKey: keys[0]}, nil
}
