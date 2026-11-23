package request

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"e-bank/backend/internal/antifraud/models"
)

func ParseGraphQuery(values url.Values) (models.GraphQuery, error) {
	var query models.GraphQuery
	for _, field := range []string{"checkId", "rootType", "rootId", "from", "to", "depth", "limit"} {
		if len(values[field]) > 1 {
			return query, fmt.Errorf("Параметр %s должен быть указан один раз.", field)
		}
	}

	if values.Has("checkId") {
		if values.Has("rootType") || values.Has("rootId") {
			return query, errors.New("Задайте checkId или пару rootType/rootId.")
		}
		id, err := strconv.ParseInt(values.Get("checkId"), 10, 32)
		if err != nil || id <= 0 {
			return query, errors.New("checkId должен быть положительным целым числом типа int32.")
		}
		query.CheckID = int32(id)
	} else {
		root, err := parseGraphRoot(values.Get("rootType"), values.Get("rootId"))
		if err != nil {
			return query, err
		}
		query.Root = root
	}

	if values.Has("from") || values.Has("to") {
		from, fromErr := time.Parse(time.RFC3339Nano, values.Get("from"))
		to, toErr := time.Parse(time.RFC3339Nano, values.Get("to"))
		if fromErr != nil || toErr != nil || !from.Before(to) || to.Sub(from) > 31*24*time.Hour {
			return query, errors.New("Задайте from и to в формате RFC 3339 и период от 0 до 31 суток.")
		}
		query.Window = &models.GraphWindow{From: from, To: to}
	}

	var err error
	query.Depth, err = graphBound(values, "depth", 2, 4)
	if err != nil {
		return query, err
	}
	query.Limit, err = graphBound(values, "limit", 100, 500)
	return query, err
}

func parseGraphRoot(kind, id string) (models.GraphNode, error) {
	if strings.TrimSpace(id) == "" || strings.ContainsRune(id, '\x00') {
		return models.GraphNode{}, errors.New("Задайте checkId или пару rootType/rootId.")
	}
	switch kind {
	case models.GraphUser, models.GraphDevice:
	case models.GraphAccount:
		accountID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || accountID <= 0 {
			return models.GraphNode{}, errors.New("rootId счёта должен быть положительным целым числом типа int32.")
		}
		id = strconv.FormatInt(accountID, 10)
	case models.GraphIP:
		ip, err := netip.ParseAddr(id)
		if err != nil || ip.Zone() != "" {
			return models.GraphNode{}, errors.New("rootId должен быть адресом IPv4 или IPv6.")
		}
		id = ip.String()
	default:
		return models.GraphNode{}, errors.New("rootType должен быть USER, BANK_ACCOUNT, DEVICE или IP_ADDRESS.")
	}
	return models.GraphNode{Type: kind, Value: id}, nil
}

func graphBound(values url.Values, name string, defaultValue, maxValue int) (int, error) {
	if !values.Has(name) {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(values.Get(name))
	if err != nil || value < 1 || value > maxValue {
		return 0, fmt.Errorf("%s должен быть целым числом от 1 до %d.", name, maxValue)
	}
	return value, nil
}
