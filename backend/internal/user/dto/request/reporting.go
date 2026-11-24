package request

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e-bank/backend/internal/user/models"
)

func ParseAnalytics(rawQuery string) (time.Time, time.Time, error) {
	query, err := parseReportingQuery(rawQuery, "from", "to")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	from, to, err := reportingPeriod(query)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if from == nil || to == nil || to.Sub(*from) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, fmt.Errorf("Укажите from и to для периода не более 31 суток.")
	}
	return *from, *to, nil
}

func ParseAudit(rawQuery string) (models.AuditFilter, error) {
	var filter models.AuditFilter
	query, err := parseReportingQuery(rawQuery, "from", "to", "actorId", "userId", "action", "limit", "offset")
	if err != nil {
		return filter, err
	}
	filter.From, filter.To, err = reportingPeriod(query)
	if err != nil {
		return filter, err
	}
	filter.ActorID, err = reportingText(query, "actorId")
	if err != nil {
		return filter, err
	}
	filter.UserID, err = reportingText(query, "userId")
	if err != nil {
		return filter, err
	}
	filter.Action, err = reportingText(query, "action")
	if err != nil {
		return filter, err
	}
	limit, err := reportingNumber(query, "limit", 50, 1, 100)
	if err != nil {
		return filter, err
	}
	filter.Limit = int(limit)
	filter.Offset, err = reportingNumber(query, "offset", 0, 0, 1<<63-1)
	return filter, err
}

func parseReportingQuery(rawQuery string, keys ...string) (url.Values, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("Некорректная строка запроса.")
	}
	for _, key := range keys {
		if len(query[key]) > 1 {
			return nil, fmt.Errorf("Параметр %s указывается один раз.", key)
		}
	}
	return query, nil
}

func reportingPeriod(query url.Values) (*time.Time, *time.Time, error) {
	from, err := reportingTime(query, "from")
	if err != nil {
		return nil, nil, err
	}
	to, err := reportingTime(query, "to")
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, fmt.Errorf("Начало периода должно быть раньше конца.")
	}
	return from, to, nil
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
