package request

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

type AccountList struct {
	OwnerID *string
	State   string
	Limit   int
	Offset  int64
}

func ParseAccountList(query url.Values) (AccountList, error) {
	for _, name := range []string{"ownerId", "state", "limit", "offset"} {
		if len(query[name]) > 1 {
			return AccountList{}, fmt.Errorf("параметр %s указывается один раз", name)
		}
	}

	params := AccountList{Limit: 50}
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
	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return AccountList{}, errors.New("limit должен быть целым числом от 1 до 100")
		}
		params.Limit = limit
	}
	if query.Has("offset") {
		offset, err := strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil || offset < 0 {
			return AccountList{}, errors.New("offset должен быть неотрицательным целым числом")
		}
		params.Offset = offset
	}
	return params, nil
}
