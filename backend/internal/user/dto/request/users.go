package request

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"e-bank/backend/internal/user/models"
)

func ParseUserID(value string) (string, error) {
	if !validText(value) || value == "." || value == ".." {
		return "", errors.New("userId должен быть непустой строкой UTF-8")
	}
	return value, nil
}

func ParseUserFilter(query url.Values) (models.UserFilter, error) {
	filter := models.UserFilter{Limit: 50}
	for _, name := range []string{"q", "enabled", "role", "limit", "offset"} {
		if len(query[name]) > 1 {
			return filter, fmt.Errorf("параметр %s должен быть указан один раз", name)
		}
	}
	if query.Has("q") {
		filter.Query = query.Get("q")
		if !validText(filter.Query) {
			return filter, errors.New("q должен быть непустой строкой UTF-8")
		}
	}
	if query.Has("enabled") {
		value := query.Get("enabled")
		if value != "true" && value != "false" {
			return filter, errors.New("enabled должен быть true или false")
		}
		enabled := value == "true"
		filter.Enabled = &enabled
	}
	if query.Has("role") {
		filter.Role = query.Get("role")
		if !models.IsApplicationRole(filter.Role) {
			return filter, errors.New("role должен быть CLIENT, ANALYST или ADMIN")
		}
	}
	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return filter, errors.New("limit должен быть целым числом от 1 до 100")
		}
		filter.Limit = limit
	}
	if query.Has("offset") {
		offset, err := strconv.Atoi(query.Get("offset"))
		if err != nil || offset < 0 {
			return filter, errors.New("offset должен быть неотрицательным целым числом")
		}
		filter.Offset = offset
	}
	return filter, nil
}

func validText(value string) bool {
	return utf8.ValidString(value) && strings.TrimSpace(value) != "" && !strings.ContainsRune(value, 0)
}
