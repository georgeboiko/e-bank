package request

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"e-bank/backend/internal/money/models"

	"github.com/shopspring/decimal"
)

var (
	uuidPattern          = regexp.MustCompile("(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	positiveMoneyPattern = regexp.MustCompile("^(0\\.(0[1-9]|[1-9][0-9])|[1-9][0-9]{0,16}\\.[0-9]{2})$")
)

func ParseCreateTransfer(body io.Reader, keys []string) (models.CreateTransfer, error) {
	if len(keys) != 1 || !uuidPattern.MatchString(keys[0]) {
		return models.CreateTransfer{}, errors.New("заголовок Idempotency-Key должен содержать один UUID")
	}

	var payload struct {
		FromAccountID int32   `json:"fromAccountId"`
		ToAccountID   int32   `json:"toAccountId"`
		Amount        string  `json:"amount"`
		DeviceID      *string `json:"deviceId"`
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return models.CreateTransfer{}, errors.New("проверьте поля и формат JSON")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return models.CreateTransfer{}, errors.New("тело запроса должно содержать один JSON-объект")
	}

	if payload.FromAccountID < 1 || payload.ToAccountID < 1 || payload.FromAccountID == payload.ToAccountID {
		return models.CreateTransfer{}, errors.New("укажите два разных положительных идентификатора счёта")
	}
	if !positiveMoneyPattern.MatchString(payload.Amount) {
		return models.CreateTransfer{}, errors.New("amount должен быть строкой с положительной суммой и двумя дробными знаками")
	}
	amount, err := decimal.NewFromString(payload.Amount)
	if err != nil {
		return models.CreateTransfer{}, errors.New("некорректная сумма amount")
	}
	if id := payload.DeviceID; id != nil &&
		(strings.TrimSpace(*id) == "" || strings.ContainsRune(*id, '\x00')) {
		return models.CreateTransfer{}, errors.New("deviceId должен содержать непробельный символ и не содержать нулевой байт")
	}

	return models.CreateTransfer{
		FromAccountID:  payload.FromAccountID,
		ToAccountID:    payload.ToAccountID,
		Amount:         amount,
		IdempotencyKey: keys[0],
		DeviceID:       payload.DeviceID,
	}, nil
}
