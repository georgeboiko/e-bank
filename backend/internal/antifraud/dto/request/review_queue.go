package request

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"e-bank/backend/internal/antifraud/models"
)

func ParseCheckList(query url.Values) (models.CheckListFilter, error) {
	window, err := parseListWindow(query, "transferId", "userId", "fromAccountId",
		"toAccountId", "automaticDecision", "minScore", "maxScore")
	if err != nil {
		return models.CheckListFilter{}, err
	}
	filter := models.CheckListFilter{From: window.from, To: window.to, Limit: window.limit, Offset: window.offset}
	if filter.TransferID, err = entityIDFilter(query, "transferId"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.FromAccountID, err = entityIDFilter(query, "fromAccountId"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.ToAccountID, err = entityIDFilter(query, "toAccountId"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.UserID, err = userIDFilter(query, "userId"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.AutomaticDecision, err = enumFilter(query, "automaticDecision", "APPROVED", "REVIEW_REQUIRED", "REJECTED"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.MinScore, err = scoreFilter(query, "minScore"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.MaxScore, err = scoreFilter(query, "maxScore"); err != nil {
		return models.CheckListFilter{}, err
	}
	if filter.MinScore != nil && filter.MaxScore != nil && *filter.MinScore > *filter.MaxScore {
		return models.CheckListFilter{}, errors.New("minScore не должен превышать maxScore")
	}
	return filter, nil
}

func ParseReviewCaseList(query url.Values) (models.ReviewCaseListFilter, error) {
	window, err := parseListWindow(query, "status", "analystId")
	if err != nil {
		return models.ReviewCaseListFilter{}, err
	}
	status, err := enumFilter(query, "status", "OPEN", "IN_PROGRESS", "CLOSED")
	if err != nil {
		return models.ReviewCaseListFilter{}, err
	}
	analystID, err := userIDFilter(query, "analystId")
	if err != nil {
		return models.ReviewCaseListFilter{}, err
	}
	return models.ReviewCaseListFilter{
		Status: status, AnalystID: analystID, From: window.from, To: window.to,
		Limit: window.limit, Offset: window.offset,
	}, nil
}

func ParseEntityID(name, value string) (int32, error) {
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil || number < 1 {
		return 0, fmt.Errorf("%s должен быть положительным целым числом типа int32", name)
	}
	return int32(number), nil
}

type listWindow struct {
	from   *time.Time
	to     *time.Time
	limit  int
	offset int64
}

func parseListWindow(query url.Values, filters ...string) (listWindow, error) {
	for _, name := range append(filters, "from", "to", "limit", "offset") {
		if len(query[name]) > 1 {
			return listWindow{}, fmt.Errorf("параметр %s указывается один раз", name)
		}
	}

	params := listWindow{limit: 50}
	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return listWindow{}, errors.New("limit должен быть целым числом от 1 до 100")
		}
		params.limit = limit
	}
	if query.Has("offset") {
		offset, err := strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil || offset < 0 {
			return listWindow{}, errors.New("offset должен быть неотрицательным целым числом")
		}
		params.offset = offset
	}

	var err error
	if params.from, err = timeFilter(query, "from"); err != nil {
		return listWindow{}, err
	}
	if params.to, err = timeFilter(query, "to"); err != nil {
		return listWindow{}, err
	}
	if params.from != nil && params.to != nil && !params.from.Before(*params.to) {
		return listWindow{}, errors.New("from должен быть раньше to")
	}
	return params, nil
}

func entityIDFilter(query url.Values, name string) (*int32, error) {
	if !query.Has(name) {
		return nil, nil
	}
	id, err := ParseEntityID(name, query.Get(name))
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func userIDFilter(query url.Values, name string) (*string, error) {
	if !query.Has(name) {
		return nil, nil
	}
	id := query.Get(name)
	if id == "" || !utf8.ValidString(id) || strings.ContainsRune(id, '\x00') {
		return nil, fmt.Errorf("%s должен быть непустой строкой UTF-8 без нулевого байта", name)
	}
	return &id, nil
}

func enumFilter(query url.Values, name string, options ...string) (string, error) {
	if !query.Has(name) {
		return "", nil
	}
	value := query.Get(name)
	if !slices.Contains(options, value) {
		return "", fmt.Errorf("%s должен быть одним из: %s", name, strings.Join(options, ", "))
	}
	return value, nil
}

func scoreFilter(query url.Values, name string) (*int32, error) {
	if !query.Has(name) {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(query.Get(name), 10, 32)
	if err != nil || parsed < 0 || parsed > 100 {
		return nil, fmt.Errorf("%s должен быть целым числом от 0 до 100", name)
	}
	score := int32(parsed)
	return &score, nil
}

func timeFilter(query url.Values, name string) (*time.Time, error) {
	if !query.Has(name) {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, query.Get(name))
	if err != nil {
		return nil, fmt.Errorf("%s должен быть датой и временем в формате RFC 3339", name)
	}
	return &value, nil
}
