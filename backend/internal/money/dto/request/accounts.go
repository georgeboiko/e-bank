package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ParseAccountState(body io.Reader) (string, error) {
	var payload struct {
		State string `json:"state"`
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return "", errors.New("проверьте поля и формат JSON")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return "", errors.New("тело запроса должно содержать один JSON-объект")
	}
	switch payload.State {
	case "ACTIVE", "BLOCKED", "CLOSED":
		return payload.State, nil
	default:
		return "", errors.New("state должен быть ACTIVE, BLOCKED или CLOSED")
	}
}

type AccountList struct {
	OwnerID *string
	State   string
	Limit   int
	Offset  int64
}

func ParseAccountList(query url.Values) (AccountList, error) {
	if err := validateSingleParams(query, "ownerId", "state", "limit", "offset"); err != nil {
		return AccountList{}, err
	}

	var params AccountList
	if query.Has("ownerId") {
		ownerID := query.Get("ownerId")
		if ownerID == "" || !utf8.ValidString(ownerID) || strings.ContainsRune(ownerID, '\x00') {
			return AccountList{}, errors.New("ownerId должен быть непустой строкой UTF-8 без нулевого байта")
		}
		params.OwnerID = &ownerID
	}
	if query.Has("state") {
		params.State = query.Get("state")
		switch params.State {
		case "ACTIVE", "BLOCKED", "CLOSED":
		default:
			return AccountList{}, errors.New("state должен быть ACTIVE, BLOCKED или CLOSED")
		}
	}
	var err error
	params.Limit, params.Offset, err = parsePagination(query)
	if err != nil {
		return AccountList{}, err
	}
	return params, nil
}

func validateSingleParams(query url.Values, names ...string) error {
	for _, name := range names {
		if len(query[name]) > 1 {
			return fmt.Errorf("параметр %s указывается один раз", name)
		}
	}
	return nil
}

func parsePagination(query url.Values) (int, int64, error) {
	limit := 50
	var offset int64
	var err error
	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, errors.New("limit должен быть целым числом от 1 до 100")
		}
	}
	if query.Has("offset") {
		offset, err = strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil || offset < 0 {
			return 0, 0, errors.New("offset должен быть неотрицательным целым числом")
		}
	}
	return limit, offset, nil
}
