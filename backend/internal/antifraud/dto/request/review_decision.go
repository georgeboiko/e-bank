package request

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type ReviewDecision struct {
	Decision string  `json:"decision"`
	Comment  *string `json:"comment"`
}

func ParseReviewDecision(body io.Reader) (ReviewDecision, error) {
	var input ReviewDecision
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return ReviewDecision{}, errors.New("проверьте поля и формат JSON")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return ReviewDecision{}, errors.New("тело запроса должно содержать один JSON-объект")
	}
	if input.Decision != "APPROVED" && input.Decision != "REJECTED" {
		return ReviewDecision{}, errors.New("decision должен быть APPROVED или REJECTED")
	}
	if input.Comment != nil && strings.ContainsRune(*input.Comment, '\x00') {
		return ReviewDecision{}, errors.New("comment не должен содержать нулевой байт")
	}
	return input, nil
}
