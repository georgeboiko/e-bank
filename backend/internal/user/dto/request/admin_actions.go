package request

import (
	"encoding/json"
	"errors"
	"io"
	"slices"

	"e-bank/backend/internal/user/models"
)

func ParseUserStatus(body io.Reader) (bool, error) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeAction(body, &input); err != nil {
		return false, err
	}
	if input.Enabled == nil {
		return false, errors.New("укажите enabled типа boolean")
	}
	return *input.Enabled, nil
}

func ParseUserRoles(body io.Reader) ([]string, error) {
	var input struct {
		Roles []string `json:"roles"`
	}
	if err := decodeAction(body, &input); err != nil {
		return nil, err
	}
	if input.Roles == nil {
		return nil, errors.New("укажите массив roles")
	}
	slices.Sort(input.Roles)
	for i, role := range input.Roles {
		if !models.IsApplicationRole(role) || (i > 0 && input.Roles[i-1] == role) {
			return nil, errors.New("roles должен содержать уникальные роли CLIENT, ANALYST или ADMIN")
		}
	}
	return input.Roles, nil
}

func decodeAction(body io.Reader, result any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return errors.New("проверьте поля и формат JSON")
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return errors.New("тело запроса должно содержать один JSON-объект")
	}
	return nil
}
