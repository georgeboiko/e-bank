package request

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e-bank/backend/internal/antifraud/models"
)

func ParseAnalytics(rawQuery string) (from time.Time, to time.Time, topLimit int, err error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return from, to, 0, fmt.Errorf("Некорректная строка запроса.")
	}
	if err := reportingSingleValues(query, "from", "to", "topLimit"); err != nil {
		return from, to, 0, err
	}
	start, err := reportingTime(query, "from")
	if err != nil {
		return from, to, 0, err
	}
	end, err := reportingTime(query, "to")
	if err != nil {
		return from, to, 0, err
	}
	if start == nil || end == nil || !start.Before(*end) || end.Sub(*start) > 31*24*time.Hour {
		return from, to, 0, fmt.Errorf("Нужны from < to и период не более 31 суток.")
	}
	limit, err := reportingNumber(query, "topLimit", 10, 1, 50)
	return *start, *end, int(limit), err
}

func ParseAudit(rawQuery string, checkID *int32) (models.AuditFilter, error) {
	filter := models.AuditFilter{CheckID: checkID}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return filter, fmt.Errorf("Некорректная строка запроса.")
	}
	if err := reportingSingleValues(query, "limit", "offset", "from", "to", "actorId", "action", "checkId"); err != nil {
		return filter, err
	}
	limit, err := reportingNumber(query, "limit", 50, 1, 100)
	if err != nil {
		return filter, err
	}
	filter.Limit = int(limit)
	filter.Offset, err = reportingNumber(query, "offset", 0, 0, 1<<63-1)
	if err != nil || checkID != nil {
		return filter, err
	}

	filter.From, err = reportingTime(query, "from")
	if err != nil {
		return filter, err
	}
	filter.To, err = reportingTime(query, "to")
	if err != nil {
		return filter, err
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return filter, fmt.Errorf("Начало периода должно быть раньше конца.")
	}
	filter.ActorID, err = reportingText(query, "actorId")
	if err != nil {
		return filter, err
	}
	filter.Action, err = reportingText(query, "action")
	if err == nil && query.Has("checkId") {
		id, parseErr := reportingNumber(query, "checkId", 0, 1, 1<<31-1)
		checkID := int32(id)
		filter.CheckID, err = &checkID, parseErr
	}
	return filter, err
}

func reportingSingleValues(query url.Values, keys ...string) error {
	for _, key := range keys {
		if len(query[key]) > 1 {
			return fmt.Errorf("Параметр %s указывается один раз.", key)
		}
	}
	return nil
}

func reportingTime(query url.Values, key string) (*time.Time, error) {
	if !query.Has(key) {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339Nano, query.Get(key))
	if err != nil {
		return nil, fmt.Errorf("Параметр %s должен содержать дату и время с часовым поясом.", key)
	}
	return &value, nil
}

func reportingNumber(query url.Values, key string, fallback, minValue, maxValue int64) (int64, error) {
	if !query.Has(key) {
		return fallback, nil
	}
	value, err := strconv.ParseInt(query.Get(key), 10, 64)
	if err != nil || value < minValue || value > maxValue {
		return 0, fmt.Errorf("Параметр %s имеет недопустимое значение.", key)
	}
	return value, nil
}

func reportingText(query url.Values, key string) (*string, error) {
	if !query.Has(key) {
		return nil, nil
	}
	value := query.Get(key)
	if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return nil, fmt.Errorf("Параметр %s должен быть непустой строкой UTF-8.", key)
	}
	return &value, nil
}
