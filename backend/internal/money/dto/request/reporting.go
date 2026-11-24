package request

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e-bank/backend/internal/money/models"
)

func ParseAnalytics(query url.Values) (time.Time, time.Time, error) {
	if err := validateSingleParams(query, "from", "to"); err != nil {
		return time.Time{}, time.Time{}, err
	}
	from, err := parseHistoryTime(query, "from")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parseHistoryTime(query, "to")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if from == nil || to == nil || !from.Before(*to) || to.Sub(*from) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, errors.New("укажите from и to: from должен быть раньше to, период не более 31 суток")
	}
	return *from, *to, nil
}

func ParseAudit(query url.Values) (models.AuditFilter, error) {
	var filter models.AuditFilter
	if err := validateSingleParams(query, "actorId", "accountId", "action"); err != nil {
		return filter, err
	}
	history, err := ParseHistory(query)
	if err != nil {
		return filter, err
	}
	filter.HistoryFilter = history
	filter.ActorID, err = parseAuditText(query, "actorId")
	if err != nil {
		return filter, err
	}
	filter.Action, err = parseAuditText(query, "action")
	if err != nil {
		return filter, err
	}
	if query.Has("accountId") {
		id, err := strconv.ParseInt(query.Get("accountId"), 10, 32)
		if err != nil || id < 1 {
			return filter, errors.New("accountId должен быть положительным целым числом типа int32")
		}
		accountID := int32(id)
		filter.AccountID = &accountID
	}
	return filter, nil
}

func parseAuditText(query url.Values, name string) (*string, error) {
	if !query.Has(name) {
		return nil, nil
	}
	value := query.Get(name)
	if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return nil, fmt.Errorf("%s должен быть непустой строкой UTF-8 без нулевого байта", name)
	}
	return &value, nil
}
