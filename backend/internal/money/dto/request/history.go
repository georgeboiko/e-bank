package request

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e-bank/backend/internal/money/models"

	"github.com/shopspring/decimal"
)

var moneyBoundPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,16})\.[0-9]{2}$`)

func ParseTransferList(query url.Values) (models.TransferListFilter, error) {
	var filter models.TransferListFilter
	if err := validateSingleParams(query, "accountId", "userId", "state", "type", "direction", "minAmount", "maxAmount"); err != nil {
		return filter, err
	}

	history, err := ParseHistory(query)
	if err != nil {
		return filter, err
	}
	filter.HistoryFilter = history

	if query.Has("accountId") {
		id, err := strconv.ParseInt(query.Get("accountId"), 10, 32)
		if err != nil || id < 1 {
			return filter, errors.New("accountId должен быть положительным целым числом типа int32")
		}
		accountID := int32(id)
		filter.AccountID = &accountID
	}
	if query.Has("userId") {
		userID := query.Get("userId")
		if userID == "" || !utf8.ValidString(userID) || strings.ContainsRune(userID, '\x00') {
			return filter, errors.New("userId должен быть непустой строкой UTF-8 без нулевого байта")
		}
		filter.UserID = &userID
	}
	if query.Has("state") {
		filter.State = query.Get("state")
		switch filter.State {
		case "PENDING", "FRAUD_CHECK_PENDING", "COMPLETING", "RELEASING_FUNDS", "COMPLETED", "REJECTED", "CANCELLED":
		default:
			return filter, errors.New("некорректное состояние state")
		}
	}
	if query.Has("type") {
		filter.Type = query.Get("type")
		if filter.Type != "TRANSFER" && filter.Type != "TOPUP" {
			return filter, errors.New("type должен быть TRANSFER или TOPUP")
		}
	}
	if query.Has("direction") {
		filter.Direction = query.Get("direction")
		if filter.AccountID == nil {
			return filter, errors.New("direction требует accountId")
		}
		if filter.Direction != "INCOMING" && filter.Direction != "OUTGOING" {
			return filter, errors.New("direction должен быть INCOMING или OUTGOING")
		}
	}

	filter.MinAmount, err = parseMoneyBound(query, "minAmount")
	if err != nil {
		return filter, err
	}
	filter.MaxAmount, err = parseMoneyBound(query, "maxAmount")
	if err != nil {
		return filter, err
	}
	if filter.MinAmount != nil && filter.MaxAmount != nil && filter.MinAmount.GreaterThan(*filter.MaxAmount) {
		return filter, errors.New("minAmount не должен превышать maxAmount")
	}
	return filter, nil
}

func ParseHistory(query url.Values) (models.HistoryFilter, error) {
	var filter models.HistoryFilter
	if err := validateSingleParams(query, "from", "to", "limit", "offset"); err != nil {
		return filter, err
	}
	limit, offset, err := parsePagination(query)
	if err != nil {
		return filter, err
	}
	filter.Limit, filter.Offset = limit, offset
	filter.From, err = parseHistoryTime(query, "from")
	if err != nil {
		return filter, err
	}
	filter.To, err = parseHistoryTime(query, "to")
	if err != nil {
		return filter, err
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return filter, errors.New("from должен быть раньше to")
	}

	// PostgreSQL stores microseconds. Rounding both bounds up preserves [from, to).
	for _, bound := range []*time.Time{filter.From, filter.To} {
		if bound != nil {
			*bound = bound.Add(time.Microsecond - time.Nanosecond).Truncate(time.Microsecond)
		}
	}
	return filter, nil
}

func parseHistoryTime(query url.Values, name string) (*time.Time, error) {
	if !query.Has(name) {
		return nil, nil
	}
	at, err := time.Parse(time.RFC3339Nano, query.Get(name))
	if err != nil {
		return nil, fmt.Errorf("%s должен быть датой и временем в формате RFC 3339", name)
	}
	return &at, nil
}

func parseMoneyBound(query url.Values, name string) (*decimal.Decimal, error) {
	if !query.Has(name) {
		return nil, nil
	}
	value := query.Get(name)
	if !moneyBoundPattern.MatchString(value) {
		return nil, fmt.Errorf("%s должен быть неотрицательной суммой с двумя дробными знаками", name)
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return nil, fmt.Errorf("некорректная сумма %s", name)
	}
	return &amount, nil
}
